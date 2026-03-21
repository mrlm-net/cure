package mcp

import "context"

// Resource represents a readable data source exposed to MCP clients via
// resources/list and resources/read. Each resource is identified by a unique URI.
//
// Implement this interface to expose static or dynamic data (files, database
// records, API responses, etc.) as MCP resources.
//
// Example:
//
//	type ReadmeResource struct{}
//
//	func (r *ReadmeResource) URI() string         { return "file:///README.md" }
//	func (r *ReadmeResource) Name() string        { return "README" }
//	func (r *ReadmeResource) Description() string { return "Project readme file" }
//	func (r *ReadmeResource) MIMEType() string    { return "text/markdown" }
//	func (r *ReadmeResource) Read(ctx context.Context) ([]mcp.ResourceContent, error) {
//	    data, err := os.ReadFile("README.md")
//	    if err != nil {
//	        return nil, err
//	    }
//	    return []mcp.ResourceContent{{
//	        Type: "resource",
//	        URI:  "file:///README.md",
//	        Text: string(data),
//	    }}, nil
//	}
type Resource interface {
	// URI returns the unique resource identifier.
	// Must be a valid URI string (e.g., "file:///path/to/file", "https://example.com/data").
	URI() string

	// Name returns a short human-readable label for the resource.
	Name() string

	// Description provides a human-readable explanation of the resource's contents.
	Description() string

	// MIMEType returns the MIME type of the resource content (e.g., "text/plain").
	MIMEType() string

	// Read retrieves the resource contents.
	// May return multiple ResourceContent blocks for compound resources.
	Read(ctx context.Context) ([]ResourceContent, error)
}
