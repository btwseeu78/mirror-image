package utility

import (
	"github.com/blang/semver/v4"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"regexp"
	"sort"
)

func NewRepository(repoName string) (name.Repository, error) {
	return name.NewRepository(repoName)
}

func ListRepositoryTags(repo name.Repository, kc authn.Keychain, filter string) ([]string, error) {
	var tags []string
	var err error
	if kc == nil {
		tags, err = remote.List(repo, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			return nil, err
		}
	} else {
		tags, err = remote.List(repo, remote.WithAuthFromKeychain(kc))
		if err != nil {
			return nil, err
		}
	}
	if filter == "" {
		return tags, nil
	}
	regex, err := regexp.Compile(filter)
	if err != nil {
		return nil, err
	}
	// Filter tags using the regex pattern
	var filteredTags []string
	for _, tag := range tags {
		if regex.MatchString(tag) {
			filteredTags = append(filteredTags, tag)
		}
	}
	return filteredTags, nil
}

// Pull image from repository
func PullAndPushImage(imageName, destname string, kc authn.Keychain) (v1.Image, error) {
	ref, err := name.ParseReference(imageName)
	if err != nil {
		return nil, err

	}
	if kc == nil {
		img, err := remote.Image(ref, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			return nil, err
		}
		return img, nil
	}
	img, err := remote.Image(ref, remote.WithAuthFromKeychain(kc))
	if err != nil {
		return nil, err
	}
	// Push the image to the destination repository

	return img, nil
}

// Get the tags need to  be synced
func GetMissingTags(sourceTags, destTags []string, limit int) (*[]string, error) {
	// Sort tags in descending order
	//sort.Sort(sort.Reverse(sort.StringSlice(sourceTags)))
	// Sort tags in descending order
	//sort.Sort(sort.Reverse(sort.StringSlice(destTags)))
	var syncTags []string
	for i := range sourceTags {
		if !contains(destTags, sourceTags[i]) {
			syncTags = append(syncTags, sourceTags[i])
		}
	}
	var semverTags []semver.Version
	for _, tag := range syncTags {
		v, err := semver.ParseTolerant(tag)
		if err == nil {
			semverTags = append(semverTags, v)
		}
	}
	sort.Slice(semverTags, func(i, j int) bool {
		return semverTags[i].GT(semverTags[j])
	})

	// Convert sorted semver tags back to string
	sortedTags := make([]string, len(semverTags))
	for i, v := range semverTags {
		sortedTags[i] = v.String()
	}
	if limit > 0 && limit < len(sortedTags) {
		sortedTags = sortedTags[:limit]
	}

	return &sortedTags, nil
}

func contains(tags []string, s string) bool {
	for _, t := range tags {
		if t == s {
			return true
		}
	}
	return false
}
