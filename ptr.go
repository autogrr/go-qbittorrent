package qbittorrent

// Ptr returns a pointer to v. Useful for constructing struct literals when
// fields carry pointer types such as *string, *int64, *bool, etc.
//
//	t := Torrent{Name: Ptr("My Torrent"), Progress: Ptr(0.5)}
func Ptr[T any](v T) *T { return &v }

// Deref returns the value pointed to by p, or the zero value of T if p is nil.
// Convenient for reading optional pointer fields without a nil check:
//
//	name := Deref(t.Name) // "" if nil
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
