package ason

type Mode int

const (
	Default Mode = iota

	// Cache large nodes. Use this mode when you carry out some
	// manual manipulations with the source AST tree. For example,
	// you duplicate nodes, which can create nodes that have the
	// same references to the original object in memory. Only
	// large nodes can be cached, such as specifications, types
	// and declarations.
	CacheRef

	// SkipComments will don't serialize any comments declarations
	SkipComments

	// ResolveObject identifiers to objects
	ResolveObject

	// ResolveScope file and package scope and identifiers objects
	ResolveScope

	// ResolveLoc allow to resolve start and end position for
	// all AST nodes.
	ResolveLoc
)
