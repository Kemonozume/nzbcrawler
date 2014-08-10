package data

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
)

type Release struct {
	Id       int64  `json:"id"`
	Checksum string `sql:"not null;unique" json:"-"`
	Url      string `json:"url"`
	Name     string `json:"name"`
	Time     int64  `json:"time"`
	Hits     int    `json:"hits"`
	Image    string `json:"image"`
	Nzb      string `json:"nzb_link"`
	Password string `json:"password"`
	Tags     []Tag  `gorm:"many2many:releases_tags;" json:"tags"`
}

type Tag struct {
	Id     int64  `json:"id"`
	Value  string `sql:"unique;not null" json:"value"`
	Weight int    `json:"weight"`
}

func (r *Release) HasTag(name string) bool {
	if len(r.Tags) < 1 {
		return false
	}
	for _, tag := range r.Tags {
		if tag.Value == name {
			return true
		}
	}
	return false
}

func (r *Release) AddTag(tag string) {
	tag = strings.ToLower(tag)
	tag = strings.TrimSpace(tag)

	if !r.HasTag(tag) {
		r.Tags = append(r.Tags, Tag{Value: tag})
	}
}

func (r *Release) EncodeName() {
	h := sha1.New()
	h.Write([]byte(r.Name))
	r.Checksum = hex.EncodeToString(h.Sum(nil))
}
