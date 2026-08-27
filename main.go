// Command gep downloads disclosure documents from EDINET API v2.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const edinetDocumentsURL = "https://api.edinet-fsa.go.jp/api/v2/documents"

var documentIDPattern = regexp.MustCompile(`^[A-Za-z0-9]+$`)

type options struct {
	documentType int
	outputDir    string
	timeout      time.Duration
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("gep", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "Usage: %s [options] DOCUMENT_ID\n", fs.Name())
		fmt.Fprintln(stderr, "Download a document from EDINET API v2.")
		fmt.Fprintln(stderr, "The API key is read from EDINET_API_KEY.")
		fmt.Fprintln(stderr, "\nOptions:")
		fs.PrintDefaults()
	}

	var opts options
	fs.IntVar(&opts.documentType, "type", 2, "document type: 1=XBRL etc. (ZIP), 2=PDF, 3=attachments (ZIP), 4=English files (ZIP), 5=CSV (ZIP)")
	fs.StringVar(&opts.outputDir, "output-dir", ".", "directory in which to save the downloaded file")
	fs.DurationVar(&opts.timeout, "timeout", 60*time.Second, "HTTP request timeout")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return errors.New("exactly one DOCUMENT_ID is required")
	}

	documentID := fs.Arg(0)
	if !documentIDPattern.MatchString(documentID) {
		return fmt.Errorf("invalid document ID %q: only ASCII letters and digits are allowed", documentID)
	}
	if opts.documentType < 1 || opts.documentType > 5 {
		return fmt.Errorf("invalid type %d: must be one of 1, 2, 3, 4, or 5", opts.documentType)
	}
	if opts.timeout <= 0 {
		return errors.New("timeout must be greater than zero")
	}

	apiKey := strings.TrimSpace(os.Getenv("EDINET_API_KEY"))
	if apiKey == "" {
		return errors.New("environment variable EDINET_API_KEY is not set")
	}

	extension := ".zip"
	if opts.documentType == 2 {
		extension = ".pdf"
	}
	outputPath := filepath.Join(opts.outputDir, documentID+extension)
	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	if err := download(ctx, http.DefaultClient, edinetDocumentsURL, documentID, opts.documentType, apiKey, outputPath); err != nil {
		return err
	}
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("resolve output path: %w", err)
	}
	fmt.Fprintln(stdout, absPath)
	return nil
}

func download(ctx context.Context, client *http.Client, baseURL, documentID string, documentType int, apiKey, outputPath string) error {
	endpoint := strings.TrimRight(baseURL, "/") + "/" + url.PathEscape(documentID)
	query := url.Values{}
	query.Set("type", fmt.Sprint(documentType))
	query.Set("Subscription-Key", apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("download document %s: %w", documentID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		detail := strings.TrimSpace(string(body))
		if detail == "" {
			detail = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("EDINET API returned HTTP %d: %s", resp.StatusCode, detail)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(outputPath), ".edinet-download-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempName := temp.Name()
	keep := false
	defer func() {
		_ = temp.Close()
		if !keep {
			_ = os.Remove(tempName)
		}
	}()

	header := make([]byte, 4)
	n, readErr := io.ReadFull(resp.Body, header)
	if readErr != nil && readErr != io.ErrUnexpectedEOF {
		return fmt.Errorf("read response: %w", readErr)
	}
	header = header[:n]
	valid := (documentType == 2 && strings.HasPrefix(string(header), "%PDF")) ||
		(documentType != 2 && len(header) >= 2 && header[0] == 'P' && header[1] == 'K')
	if !valid {
		return errors.New("EDINET API response is not in the expected PDF/ZIP format")
	}
	if _, err := temp.Write(header); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if _, err := io.Copy(temp, resp.Body); err != nil {
		return fmt.Errorf("save response: %w", err)
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("flush temporary file: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tempName, outputPath); err != nil {
		return fmt.Errorf("replace output file: %w", err)
	}
	keep = true
	return nil
}
