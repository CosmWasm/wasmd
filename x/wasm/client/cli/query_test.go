package cli

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	flag "github.com/spf13/pflag"
)

func paginationFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String(flags.FlagPageKey, "", "")
	fs.Uint64(flags.FlagOffset, 0, "")
	fs.Uint64(flags.FlagLimit, 100, "")
	fs.Bool(flags.FlagCountTotal, false, "")
	fs.Uint64(flags.FlagPage, 0, "")
	fs.Bool(flags.FlagReverse, false, "")
	return fs
}

func TestReadPageRequestInvalidPageKeyReturnsErrorWithoutMutation(t *testing.T) {
	fs := paginationFlagSet()
	const malformedPageKey = "%%%invalid"
	if err := fs.Set(flags.FlagPageKey, malformedPageKey); err != nil {
		t.Fatal(err)
	}

	pageReq, err := readPageRequest(fs)
	if err == nil {
		t.Fatal("expected an error for malformed page key")
	}
	if pageReq != nil {
		t.Fatalf("expected no page request on error, got %v", pageReq)
	}
	if !strings.Contains(err.Error(), "invalid --page-key") {
		t.Fatalf("expected page-key context in error, got %v", err)
	}
	got, err := fs.GetString(flags.FlagPageKey)
	if err != nil {
		t.Fatal(err)
	}
	if got != malformedPageKey {
		t.Fatalf("page-key flag was modified: got %q, want %q", got, malformedPageKey)
	}
}

func TestReadPageRequestDecodesPageKeyWithoutMutation(t *testing.T) {
	fs := paginationFlagSet()
	const rawPageKey = "opaque-pagination-key"
	encodedPageKey := base64.StdEncoding.EncodeToString([]byte(rawPageKey))
	if err := fs.Set(flags.FlagPageKey, encodedPageKey); err != nil {
		t.Fatal(err)
	}

	pageReq, err := readPageRequest(fs)
	if err != nil {
		t.Fatal(err)
	}
	if string(pageReq.Key) != rawPageKey {
		t.Fatalf("unexpected page key: got %q, want %q", pageReq.Key, rawPageKey)
	}
	got, err := fs.GetString(flags.FlagPageKey)
	if err != nil {
		t.Fatal(err)
	}
	if got != encodedPageKey {
		t.Fatalf("page-key flag was modified: got %q, want %q", got, encodedPageKey)
	}
}
