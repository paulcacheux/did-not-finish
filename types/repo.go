package types

type Repomd struct {
	Data []RepomdData `xml:"data"`
}

type RepomdData struct {
	Type         string   `xml:"type,attr"`
	Size         int      `xml:"size"`
	OpenSize     int      `xml:"open-size"`
	Location     Location `xml:"location"`
	Checksum     Checksum `xml:"checksum"`
	OpenChecksum Checksum `xml:"open-checksum"`
}

type Location struct {
	Href string `xml:"href,attr"`
}

type MetaLink struct {
	Files []MetaLinkFile `xml:"files"`
}

type MetaLinkFile struct {
	Name string `xml:"name,attr"`
}
