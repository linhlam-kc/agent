package printer

import (
	"bytes"
	"fmt"
	"testing"
	"text/tabwriter"
)

func TestTabWriter(t *testing.T) {
	t.Skip()

	var buf bytes.Buffer
	defer func() {
		fmt.Println(buf.String())
	}()

	tw := tabwriter.NewWriter(&buf, 0, 8, 1, ' ', tabwriter.DiscardEmptyColumns|tabwriter.TabIndent)
	defer tw.Flush()

	tw.Write([]byte("s\t= 5\nlongname\v= 2\na\v= c\n\n"))
	tw.Write([]byte("empty.block\v\v{}\nempty.block\v'labeled'\v{}\n\n"))
	tw.Write([]byte("empty.block\v\v{}"))
}
