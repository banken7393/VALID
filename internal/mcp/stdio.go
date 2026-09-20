package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// ServeStdio runs the MCP server over stdin/stdout (newline-delimited JSON-RPC).
func (s *Server) ServeStdio(ctx context.Context) error {
	return s.ServeStdioIO(ctx, os.Stdin, os.Stdout)
}

// ServeStdioIO runs the MCP server using the provided reader/writer pair.
func (s *Server) ServeStdioIO(ctx context.Context, in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read stdin: %w", err)
		}
		line = trimSpaceBytes(line)
		if len(line) == 0 {
			continue
		}

		resp, err := s.HandleBytes(ctx, line)
		if err != nil {
			return err
		}
		if resp == nil {
			continue
		}
		if _, err := out.Write(append(resp, '\n')); err != nil {
			return fmt.Errorf("write stdout: %w", err)
		}
	}
}

func trimSpaceBytes(b []byte) []byte {
	// Avoid importing bytes just for TrimSpace on hot path clarity.
	i := 0
	j := len(b)
	for i < j && (b[i] == ' ' || b[i] == '\t' || b[i] == '\r' || b[i] == '\n') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\t' || b[j-1] == '\r' || b[j-1] == '\n') {
		j--
	}
	return b[i:j]
}

// EncodeJSON is a small helper for tests / HTTP.
func EncodeJSON(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
