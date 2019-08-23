package types

// FullBlock carries a block header and the message and receipt collections
// referenced from the header.
type FullBlock struct {
	Header   *Block
	Messages []*SignedMessage
	Receipts []*MessageReceipt
}
