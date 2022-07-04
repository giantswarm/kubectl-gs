package filesystem

type Dir struct {
	dirs  []*Dir
	files []*File
	name  string
}

type File struct {
	data []byte
	name string
}

type CreatorConfig struct {
	Path   string
	DryRun bool
}

type Creator struct {
	path   string
	dryRun bool
}
