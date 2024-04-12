package paginator_test

import (
	"net/url"
	"testing"

	"azugo.io/core/paginator"

	"github.com/go-quicktest/qt"
)

func TestPaginatorNew(t *testing.T) {
	total := 105
	pageSize := 20
	current := 2
	paginator := paginator.New(total, pageSize, current)

	qt.Check(t, qt.Equals(paginator.Total(), total))
	qt.Check(t, qt.Equals(paginator.PageSize(), pageSize))
	qt.Check(t, qt.Equals(paginator.Current(), current))
	qt.Check(t, qt.Equals(paginator.TotalPages(), total/pageSize+1))
	qt.Check(t, qt.IsFalse(paginator.IsFirst()))
	qt.Check(t, qt.IsFalse(paginator.IsLast()))
	qt.Check(t, qt.IsTrue(paginator.HasNext()))
	qt.Check(t, qt.IsTrue(paginator.HasPrevious()))
	qt.Check(t, qt.Equals(paginator.Previous(), current-1))
	qt.Check(t, qt.Equals(paginator.Next(), current+1))
}

func TestPaginatorNewEmpty(t *testing.T) {
	total := 0
	pageSize := 0
	current := 0
	paginator := paginator.New(total, pageSize, current)

	qt.Check(t, qt.Equals(paginator.Total(), 0))
	qt.Check(t, qt.Equals(paginator.PageSize(), 1))
	qt.Check(t, qt.Equals(paginator.Current(), 1))
	qt.Check(t, qt.Equals(paginator.TotalPages(), 1))
	qt.Check(t, qt.IsTrue(paginator.IsFirst()))
	qt.Check(t, qt.IsTrue(paginator.IsLast()))
	qt.Check(t, qt.IsFalse(paginator.HasNext()))
	qt.Check(t, qt.IsFalse(paginator.HasPrevious()))
	qt.Check(t, qt.Equals(paginator.Previous(), 1))
	qt.Check(t, qt.Equals(paginator.Next(), 1))
}

func TestPaginatorCurrent(t *testing.T) {
	pag := paginator.New(1, 1, 2)
	qt.Check(t, qt.Equals(pag.Current(), 1))
}

func TestPaginatorSetURL(t *testing.T) {
	paginator := paginator.New(1, 1, 1)
	testurl, err := url.Parse("http://localhost:3000")
	qt.Assert(t, qt.IsNil(err))
	paginator.SetURL(testurl)
	qt.Check(t, qt.Equals(paginator.GetURL(), testurl))
}

func TestPaginatorLinks(t *testing.T) {
	total := 105
	pageSize := 20
	current := 2
	paginator := paginator.New(total, pageSize, current)
	testurl, err := url.Parse("http://localhost:3000")
	qt.Assert(t, qt.IsNil(err))
	paginator.SetURL(testurl)

	links := paginator.Links()
	qt.Check(t, qt.HasLen(links, 4))

	link := "<http://localhost:3000?page=3&per_page=20>; rel=\"next\""
	qt.Check(t, qt.Equals(links[0], link))
}
