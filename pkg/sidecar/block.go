package sidecar

import (
	"bytes"
	"encoding/json"
)

// Block defines model for Block.
type Block int

// HTTP and Stream are the two major Blocks in the NGINX config.
// It's very important to keep track of which Block a Port belongs in.
const (
	HTTP Block = iota
	Stream
)

// String returns a Block as a string.
func (b Block) String() string {
	return [...]string{"HTTP", "Stream"}[b]
}

// MarshalJSON marshals a Block enum into JSON.
func (b Block) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(b.String())
	buffer.WriteString(`"`)

	return buffer.Bytes(), nil
}

// UnmarshalJSON unmarshals a Block, to a Block enum.
func (b *Block) UnmarshalJSON(blockBytes []byte) error {
	var j string
	if err := json.Unmarshal(blockBytes, &j); err != nil {
		return err
	}
	*b = toBlock(j)

	return nil
}

func toBlock(str string) Block {
	blocks := map[string]Block{
		"HTTP":   HTTP,
		"Stream": Stream,
	}

	return blocks[str]
}
