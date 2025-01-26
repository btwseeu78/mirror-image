package utility

import (
	"fmt"
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

func SyncTags(sourceTags, destTags []string, chunkSize int) []string {
	// Sort tags in descending order
	sort.Sort(sort.Reverse(sort.StringSlice(sourceTags)))
	// Sort tags in descending order
	sort.Sort(sort.Reverse(sort.StringSlice(destTags)))
	var syncTags []string
	if len(sourceTags) < 5 {
		fmt.Println("Less than 5 tags found in the repository")
		for i := range sourceTags {
			if !contains(destTags, sourceTags[i]) {
				syncTags = append(syncTags, sourceTags[i])
			}
		}
	} else {
		for i := range sourceTags[:5] {
			if !contains(destTags, sourceTags[i]) {
				syncTags = append(syncTags, sourceTags[i])
			}
		}
	}
	return syncTags
}

// Pull image from repository
func PullImage(imageName string, kc authn.Keychain) (v1.Image, error) {
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

	return img, nil
}

// contains checks if a slice contains a specific element
func contains(slice []string, element string) bool {
	for _, item := range slice {
		if item == element {
			return true
		}
	}
	return false
}
