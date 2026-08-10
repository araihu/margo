package deck

type Metadata struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
}

type Slide struct {
	ordinal  int
	id       string
	markdown []byte
}

func (s Slide) Ordinal() int {
	return s.ordinal
}

func (s Slide) ID() string {
	return s.id
}

func (s Slide) Markdown() []byte {
	return append([]byte(nil), s.markdown...)
}

type Document struct {
	name     string
	metadata Metadata
	slides   []Slide
}

func (d *Document) Metadata() Metadata {
	if d == nil {
		return Metadata{}
	}
	return d.metadata
}

func (d *Document) Slides() []Slide {
	if d == nil {
		return nil
	}
	result := make([]Slide, len(d.slides))
	for index, slide := range d.slides {
		result[index] = Slide{
			ordinal:  slide.ordinal,
			id:       slide.id,
			markdown: slide.Markdown(),
		}
	}
	return result
}
