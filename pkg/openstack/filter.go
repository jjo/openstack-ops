package openstack

import (
	"regexp"
	"time"

	"golang.org/x/exp/slices"
)

type Filter interface {
	CreatedBefore(time.Time) bool
	IncludesRegex(string) bool
	ExcludesRegex(string) bool
	HasTag(string) bool
}

type OSResourceFilter struct {
	createdBefore time.Time
	incRe         *regexp.Regexp
	excRe         *regexp.Regexp
	tag           string
	tagMatch      bool
}

func (filter *OSResourceFilter) WithCreatedBefore(t time.Time) *OSResourceFilter {
	filter.createdBefore = t
	return filter
}

func (filter *OSResourceFilter) WithIncRe(re string) *OSResourceFilter {
	if re != "" {
		filter.incRe = regexp.MustCompile(re)
	}

	return filter
}

func (filter *OSResourceFilter) WithExcRe(re string) *OSResourceFilter {
	if re != "" {
		filter.excRe = regexp.MustCompile(re)
	}

	return filter
}

func (filter *OSResourceFilter) WithTagMatch(tagMatch bool) *OSResourceFilter {
	filter.tagMatch = tagMatch
	return filter
}

func (filter *OSResourceFilter) WithTag(tag string) *OSResourceFilter {
	filter.tag = tag
	if tag == "" {
		filter.tagMatch = false
	}

	return filter
}

func NewOSResourceFilter(t time.Time, incStr, excStr, tag string, tagMatch bool) *OSResourceFilter {
	filter := (&OSResourceFilter{}).
		WithCreatedBefore(t).
		WithIncRe(incStr).
		WithExcRe(excStr).
		WithTag(tag).
		WithTagMatch(tagMatch)
	log.Debugf("filter=%#v", filter)
	return filter
}

func (filter *OSResourceFilter) Run(r OSResourceInterface) bool {
	strAll := r.StringAll()
	ret := r.CreatedBefore(filter.createdBefore) &&
		(filter.incRe == nil || filter.incRe.MatchString(strAll)) &&
		(filter.excRe == nil || !filter.excRe.MatchString(strAll)) &&
		(!filter.tagMatch || slices.Contains(r.GetTags(), filter.tag))
	log.Debugf("filter.Run(): strAll -> %v, ret: %v", strAll, filter, ret)
	return ret
}
