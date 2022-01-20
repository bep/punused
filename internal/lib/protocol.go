package lib

import lsp "github.com/sourcegraph/go-lsp"

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

// InitializedParams missing in github.com/sourcegraph/go-lsp borrowed from https://github.com/golang/tools/tree/master/gopls
// TODO(bep) consolidate with lsp/lsp.go
type InitializedParams struct{}

type SymbolTag float64
