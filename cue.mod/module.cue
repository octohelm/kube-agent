module: "github.com/octohelm/kube-agent"

require: {
	"github.com/octohelm/cuem": "v0.0.0-20210407103607-6920f6851876"
	"k8s.io/api":               "v0.21.0" @indirect()
	"k8s.io/apimachinery":      "v0.21.0" @indirect()
}
