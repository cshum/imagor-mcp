package tools

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// previewBytes caps what is attached to a tool result. A composition can be
// arbitrarily large - full-resolution sources are common - and a tool result
// travels back into the caller's context, so an uncapped image is a way to
// spend someone else's budget without asking.
const previewBytes = 1 << 20 // 1 MiB

// previewMaxEdge is the longest edge of the preview. Images cost tokens by area
// rather than by bytes, and a caller comparing a layout does not need the full
// render to see that a panel is in the wrong place. A 1800x942 result scaled to
// fit 768 is roughly a quarter of the tokens and still legible for placement.
const previewMaxEdge = 768

// previewTimeout is separate from verifyTimeout: fetching the image is a second
// request, and a slow render should not consume the budget for the report.
const previewTimeout = 30 * time.Second

// previewImage fetches the rendered result and returns it as base64 for an MCP
// image content block, so a caller that can see images can compare the result
// against what it meant to build.
//
// The preview is requested through a scaled path rather than the full render.
// Images cost tokens by area, so attaching an 1800x942 result at full size costs
// several times what the same layout costs at 768 on the long edge, and the
// difference between the two is not visible when the question is whether a panel
// sits in the right place.
//
// Returns nil when there is nothing useful to attach, with the reason in note:
// a caller that cannot see images loses nothing, and one that can is told why it
// got no picture rather than being left to wonder. Never an error - a preview is
// a convenience on top of a report that has already succeeded, and failing the
// whole call because the picture did not arrive would be the wrong trade.
func (c Config) previewImage(ctx context.Context, imageURL string) ([]byte, string, string) {
	if imageURL == "" {
		return nil, "", ""
	}
	reqCtx, cancel := context.WithTimeout(ctx, previewTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, "", fmt.Sprintf("could not request a preview: %v", err)
	}
	// Ask only for what will be attached. A server that honours this sends fewer
	// bytes than the cap; one that ignores it is caught by the read limit below.
	req.Header.Set("Accept", "image/png,image/jpeg,image/*")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Sprintf("could not fetch a preview: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Sprintf("no preview: the image request returned %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, previewBytes+1))
	if err != nil {
		return nil, "", fmt.Sprintf("could not read a preview: %v", err)
	}
	if len(body) > previewBytes {
		return nil, "", fmt.Sprintf(
			"no preview: the result is larger than %d KiB. Ask for a smaller render to see it",
			previewBytes>>10)
	}
	mime := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	if mime == "" || !strings.HasPrefix(mime, "image/") {
		// Sniff the common cases rather than trusting a header, because a result
		// served without one would otherwise be attached under a wrong type.
		switch {
		case bytes.HasPrefix(body, []byte("\x89PNG")):
			mime = "image/png"
		case bytes.HasPrefix(body, []byte("\xff\xd8\xff")):
			mime = "image/jpeg"
		case bytes.HasPrefix(body, []byte("GIF8")):
			mime = "image/gif"
		case bytes.HasPrefix(body, []byte("RIFF")) && len(body) > 12 && bytes.Equal(body[8:12], []byte("WEBP")):
			mime = "image/webp"
		default:
			return nil, "", "no preview: the result is not an image"
		}
	}
	return body, mime, ""
}

// withPreview attaches the rendered result to a tool result that already carries
// a text report, so a caller reads the outcome and sees the picture in one call.
//
// Content is set explicitly rather than left for the SDK, which would otherwise
// fill it with the JSON of the structured output alone.
func withPreview(out VerifyOutput, data []byte, mime, note string, scale int) *mcp.CallToolResult {
	attached := len(data) > 0
	res := &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: verifySummary(out, note, attached, scale)}},
	}
	if attached {
		// Data is the raw image: the wire format base64-encodes a []byte field
		// already, so encoding it here would base64 it twice and a client would
		// decode once and still hold base64 rather than pixels.
		res.Content = append(res.Content, &mcp.ImageContent{
			Data:     data,
			MIMEType: mime,
		})
	}
	return res
}

// verifySummary is the text half of a verified result. It leads with what was
// observed so a caller that cannot see the attached image still gets the report,
// and names the preview so one that can knows what it is looking at.
func verifySummary(out VerifyOutput, note string, attached bool, scale int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "status %d", out.Status)
	if out.Width > 0 && out.Height > 0 {
		fmt.Fprintf(&b, ", result %dx%d", out.Width, out.Height)
	}
	if len(out.Results) > 0 {
		var applied, declined []string
		for _, r := range out.Results {
			switch r.Outcome {
			case OutcomeApplied:
				applied = append(applied, r.Name)
			case OutcomeDeclined:
				declined = append(declined, r.Name)
			}
		}
		if len(applied) > 0 {
			fmt.Fprintf(&b, "\napplied: %s", strings.Join(applied, ", "))
		}
		if len(declined) > 0 {
			fmt.Fprintf(&b, "\ndeclined: %s", strings.Join(declined, ", "))
		}
	}
	if note != "" {
		fmt.Fprintf(&b, "\n%s", note)
	}
	fmt.Fprintf(&b, "\nurl: %s", out.AbsoluteURL)
	if attached {
		b.WriteString("\nThe rendered result is attached.")
		if scale > 0 {
			// Without the scale the caller cannot tell which coordinate system it
			// is looking at, and a region asked for in preview pixels lands
			// somewhere else entirely in the result.
			fmt.Fprintf(&b, " It is %d%% of the result, so a detail too small to read "+
				"here can be seen by calling again with preview_region as x,y,w,h in "+
				"result pixels.", scale)
		}
		b.WriteString(" Compare it against your reference.")
	}
	return b.String()
}
