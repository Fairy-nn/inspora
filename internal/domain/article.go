package domain

type Article struct {
	Title   string
	Content string
	Author  Author
}

type Author struct {
	ID   int64
	Name string
}
