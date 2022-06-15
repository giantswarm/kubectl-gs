package structure

type McConfig struct {
	Name            string
	RefreshInterval string
	RefreshTimeout  string
	RepositoryKind  string
	RepositoryName  string
	ServiceAccount  string
}

type Dir struct {
	dirs  []*Dir
	files []*File
	name  string
}

type File struct {
	data []byte
	name string
}

func newDir(name string) *Dir {
	return &Dir{
		dirs:  make([]*Dir, 0),
		files: make([]*File, 0),
		name:  name,
	}
}
