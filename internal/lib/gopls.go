package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	lsp "github.com/sourcegraph/go-lsp"
	"golang.org/x/sync/errgroup"
)

// Used in tests.
var (
	UnusedVar         = "unused"
	UsedVar           = "used"
	OnlyUsedInTestVar = "only used in test"
)

type GoplsClient struct {
	workspaceDir string

	callMu sync.Mutex
	conn   Conn
}

func (c *GoplsClient) Close() error {
	return c.conn.Close()
}

func NewClient(workspaceDir string) (*GoplsClient, error) {
	workspaceDir = path.Clean(filepath.ToSlash(workspaceDir))

	args := []string{"serve"} //, "-rpc.trace", "-logfile=/Users/bep/dev/gopls.log"}
	cmd := exec.Command("gopls", args...)
	cmd.Stderr = os.Stderr
	conn, err := newConn(cmd)
	if err != nil {
		return nil, err
	}

	if err := conn.Start(); err != nil {
		return nil, err
	}

	client := &GoplsClient{conn: conn, workspaceDir: workspaceDir}

	initParams := &lsp.InitializeParams{
		RootURI: lsp.DocumentURI(client.documentURI("")),
		Capabilities: lsp.ClientCapabilities{
			TextDocument: lsp.TextDocumentClientCapabilities{
				DocumentSymbol: struct {
					SymbolKind struct {
						ValueSet []int `json:"valueSet,omitempty"`
					} `json:"symbolKind,omitEmpty"`

					HierarchicalDocumentSymbolSupport bool `json:"hierarchicalDocumentSymbolSupport,omitempty"`
				}{
					HierarchicalDocumentSymbolSupport: true,
				},
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.Initialize(ctx, initParams)
	if err != nil {
		return nil, err
	}

	err = client.Initialized(ctx)

	return client, err
}

func (s *GoplsClient) documentURI(filename string) string {
	filename = filepath.ToSlash(filename)
	if path.IsAbs(filename) {
		return "file://" + filename
	}
	return "file://" + path.Join(s.workspaceDir, filename)
}

func (s *GoplsClient) Initialize(ctx context.Context, params *lsp.InitializeParams) (*lsp.InitializeResult, error) {
	var result lsp.InitializeResult

	if err := s.Call(ctx, "initialize", params, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *GoplsClient) Initialized(ctx context.Context) error {
	return s.Call(ctx, "initialized", &InitializedParams{}, nil)
}

func (s *GoplsClient) DocumentSymbol(ctx context.Context, filename string) ([]*Symbol, error) {
	uri := lsp.DocumentURI(s.documentURI(filename))
	params := &lsp.DocumentSymbolParams{
		TextDocument: lsp.TextDocumentIdentifier{
			URI: uri,
		},
	}

	var result []map[string]interface{}

	if err := s.Call(ctx, "textDocument/documentSymbol", params, &result); err != nil {
		return nil, err
	}

	var symbols []*Symbol
	for _, m := range result {
		s, err := mapToSymbol(uri, m)
		if err != nil {
			return nil, err
		}
		symbols = append(symbols, s)
	}

	return symbols, nil
}

func (s *GoplsClient) DocumentReferences(ctx context.Context, loc lsp.Location) ([]*lsp.Location, error) {
	start := loc.Range.Start

	params := &lsp.ReferenceParams{
		Context: lsp.ReferenceContext{
			IncludeDeclaration: false,
		},
		TextDocumentPositionParams: lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{
				URI: loc.URI,
			},
			Position: start,
		},
	}

	var result []*lsp.Location

	if err := s.Call(ctx, "textDocument/references", params, &result); err != nil {
		return nil, err
	}
	return result, nil
}

var requestID uint64 = 5000

// Call calls the gopls method with the params given. If result is non-nil, the response body is unmarshalled into it.
func (c *GoplsClient) Call(ctx context.Context, method string, params, result interface{}) error {
	// Only allow one call at a time for now.
	c.callMu.Lock()
	defer c.callMu.Unlock()

	id := atomic.AddUint64(&requestID, 1)
	req := request{
		RPCVersion: "2.0",
		ID: lsp.ID{
			Num: id,
		},
		Method: method,
		Params: params,
	}

	if err := c.Write(req); err != nil {
		return err
	}

	respChan := make(chan response)
	wg, ctx := errgroup.WithContext(ctx)
	wg.Go(func() error {
		return c.Read(ctx, id, respChan)
	})

	var unmarshalErr error

	select {
	case resp := <-respChan:
		if result != nil && resp.Result != nil {
			unmarshalErr = json.Unmarshal(resp.Result, result)
		}
	case <-ctx.Done():
		return ctx.Err()
	}

	if err := wg.Wait(); err != nil {
		return err
	}
	return unmarshalErr
}

// Write writes a request to gopls using the format specified by:
// https://github.com/Microsoft/language-server-protocol/blob/gh-pages/_specifications/specification-3-14.md#text-documents
func (c *GoplsClient) Write(r request) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = c.conn.Write([]byte(fmt.Sprintf("Content-Length: %d\r\n\r\n", len(b))))
	if err != nil {
		return err
	}
	_, err = c.conn.Write(b)
	return err
}

// Read reads from gopls until candidate is found and sent on respChan.
func (c *GoplsClient) Read(ctx context.Context, candidate uint64, respChan chan<- response) error {
	done := make(chan bool)
	var wg errgroup.Group

	wg.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				buff := make([]byte, 16)
				_, err := io.ReadFull(c.conn, buff)
				if err != nil {
					return err
				}

				cl := make([]byte, 0, 2)
				buff = buff[:1]
				for {
					_, err := io.ReadFull(c.conn, buff)
					if err != nil {
						return err
					}
					if buff[0] == '\r' {
						break
					}
					cl = append(cl, buff[0])
				}

				// Consume the \n\r\n
				buff = buff[:3]
				io.ReadFull(c.conn, buff)

				contentLength, err := strconv.Atoi(string(cl))
				if err != nil {
					return err
				}

				buff = make([]byte, contentLength)
				_, err = io.ReadFull(c.conn, buff)
				if err != nil {
					return err
				}

				var resp response
				if err := json.Unmarshal(buff, &resp); err != nil {
					return err
				}

				// gopls sends a lot of chatter with ID=0 (notifications meant for the editor).
				// We need to ignore those.
				if resp.ID == candidate {
					close(done)
					respChan <- resp
					return nil
				}
			}
		}
	})

	wg.Go(func() error {
		for {
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return c.conn.Close()
			}
		}
	})

	return wg.Wait()
}

type request struct {
	RPCVersion string      `json:"jsonrpc"`
	ID         lsp.ID      `json:"id"`
	Method     string      `json:"method"`
	Params     interface{} `json:"params"`
}

type response struct {
	RPCVersion string          `json:"jsonrpc"`
	ID         uint64          `json:"id"`
	Result     json.RawMessage `json:"result"`
}

func mapToSymbol(uri lsp.DocumentURI, m map[string]interface{}) (*Symbol, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	var ds DocumentSymbol
	if err = json.Unmarshal(b, &ds); err != nil {
		return nil, err
	}

	return &Symbol{
		Name: ds.Name,
		Kind: ds.Kind,
		Location: lsp.Location{
			URI:   uri,
			Range: ds.SelectionRange,
		},
	}, nil
}

func newConn(cmd *exec.Cmd) (_ Conn, err error) {
	in, err := cmd.StdinPipe()
	if err != nil {
		return Conn{}, err
	}
	defer func() {
		if err != nil {
			in.Close()
		}
	}()

	out, err := cmd.StdoutPipe()
	c := Conn{out, in, cmd}

	return c, err
}

type Conn struct {
	io.ReadCloser
	io.WriteCloser
	cmd *exec.Cmd
}

// Start starts conn's Cmd.
func (c Conn) Start() error {
	err := c.cmd.Start()
	if err != nil {
		return c.Close()
	}
	return err
}

// Close closes conn's WriteCloser ReadClosers.
func (c Conn) Close() error {
	writeErr := c.WriteCloser.Close()
	readErr := c.ReadCloser.Close()

	if writeErr != nil {
		return writeErr
	}

	return readErr
}

type Symbol struct {
	Name     string
	Kind     lsp.SymbolKind
	Location lsp.Location
}

// Types missing in github.com/sourcegraph/go-lsp borrowed from https://github.com/golang/tools/tree/master/gopls
// TODO(bep) consolidate with lsp/lsp.go
type InitializedParams struct{}

/**
 * Represents programming constructs like variables, classes, interfaces etc.
 * that appear in a document. Document symbols can be hierarchical and they
 * have two ranges: one that encloses its definition and one that points to
 * its most interesting range, e.g. the range of an identifier.
 */
type DocumentSymbol struct {
	/**
	 * The name of this symbol. Will be displayed in the user interface and therefore must not be
	 * an empty string or a string only consisting of white spaces.
	 */
	Name string `json:"name"`
	/**
	 * More detail for this symbol, e.g the signature of a function.
	 */
	Detail string `json:"detail,omitempty"`
	/**
	 * The kind of this symbol.
	 */
	Kind lsp.SymbolKind `json:"kind"`
	/**
	 * Tags for this document symbol.
	 *
	 * @since 3.16.0
	 */
	Tags []SymbolTag `json:"tags,omitempty"`
	/**
	 * Indicates if this symbol is deprecated.
	 *
	 * @deprecated Use tags instead
	 */
	Deprecated bool `json:"deprecated,omitempty"`
	/**
	 * The range enclosing this symbol not including leading/trailing whitespace but everything else
	 * like comments. This information is typically used to determine if the the clients cursor is
	 * inside the symbol to reveal in the symbol in the UI.
	 */
	Range lsp.Range `json:"range"`
	/**
	 * The range that should be selected and revealed when this symbol is being picked, e.g the name of a function.
	 * Must be contained by the the `range`.
	 */
	SelectionRange lsp.Range `json:"selectionRange"`
	/**
	 * Children of this symbol, e.g. properties of a class.
	 */
	Children []DocumentSymbol `json:"children,omitempty"`
}

type SymbolTag float64
