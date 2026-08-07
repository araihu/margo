package authority

import "fmt"

// verifyReceiptIdentity keeps transport provenance separate from the
// self-digesting authority record while still binding source and transport.
func verifyReceiptIdentity(record, source AuthorityReceipt) error {
	if record.Transport != source.Transport || record.Source != source.Source {
		return fmt.Errorf("authority.receipt_identity_mismatch")
	}
	return nil
}
