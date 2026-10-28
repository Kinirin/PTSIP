package releasecontroller

// Publication evidence cannot be admitted while successor authority is unresolved.
func AdmitPublicationEvidenceAuthority(root string) error { return AdmitCTRLPublicationPolicy(root) }
