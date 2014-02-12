package data

type Release struct {
	Checksum string `json:"checksum"`
	Url      string `json:"url"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Time     int64  `json:"time"`
	Rating   int32  `json:"rating"`
	Hits     int    `json:"hits"`
	Image    string `db:"-"`
}

func (r Release) AddTag(tag string, pr *Release) {
	if pr.Tag == "" {
		pr.Tag = tag
	} else {
		pr.Tag += "," + tag
	}
}
