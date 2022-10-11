package main

type Metadata struct {
	Packages []Package `xml:"package"`
}

type Package struct {
	Type     string   `xml:"type,attr"`
	Name     string   `xml:"name"`
	Arch     string   `xml:"arch"`
	Checksum Checksum `xml:"checksum"`
}

type Version struct {
	Epoch string `xml:"epoch,attr"`
	Ver   string `xml:"ver,attr"`
	Rel   string `xml:"rel,attr"`
}

type Checksum struct {
	Hash string `xml:",chardata"`
	Type string `xml:"type,attr"`
}
