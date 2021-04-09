package reflectutil

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestNewTagValue(t *testing.T) {
	t.Run("Tag", func(t *testing.T) {
		tv := NewTagValue("name,opt=v,opt2,k=test,k=1")

		n, _ := tv.Name()
		gomega.NewWithT(t).Expect(n).To(gomega.Equal("name"))
		gomega.NewWithT(t).Expect(tv.Flags()).To(gomega.Equal(map[string][]string{
			"opt":  {"v"},
			"k":    {"test", "1"},
			"opt2": nil,
		}))
	})
}
