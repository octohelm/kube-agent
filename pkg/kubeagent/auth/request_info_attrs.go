package auth

import (
	"k8s.io/apiserver/pkg/authentication/user"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
)

type RequestInfoAttrs struct {
	apirequest.RequestInfo
}

func (r *RequestInfoAttrs) GetUser() user.Info {
	return nil
}

func (r *RequestInfoAttrs) GetVerb() string {
	return r.Verb
}

func (r *RequestInfoAttrs) IsReadOnly() bool {
	return false
}

func (r *RequestInfoAttrs) GetNamespace() string {
	return r.Namespace
}

func (r *RequestInfoAttrs) GetResource() string {
	return r.Resource
}

func (r *RequestInfoAttrs) GetSubresource() string {
	return r.Subresource
}

func (r *RequestInfoAttrs) GetName() string {
	return r.Name
}

func (r *RequestInfoAttrs) GetAPIGroup() string {
	return r.APIGroup
}

func (r *RequestInfoAttrs) GetAPIVersion() string {
	return r.APIVersion
}

func (r *RequestInfoAttrs) IsResourceRequest() bool {
	return r.RequestInfo.IsResourceRequest
}

func (r *RequestInfoAttrs) GetPath() string {
	return r.Path
}
