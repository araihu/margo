package margo

// Source is one immutable compilation input after Compile returns.
type Source struct {
	Name    string
	Content []byte
	BaseURL string
}

func (s Source) clone() Source {
	return Source{
		Name:    s.Name,
		Content: append([]byte(nil), s.Content...),
		BaseURL: s.BaseURL,
	}
}
