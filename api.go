// Copyright 2016 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package zoekt

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/google/zoekt/query"
)

// FileMatch contains all the matches within a file.
type FileMatch struct {
	// Ranking; the higher, the better.
	Score    float64 // TODO - hide this field?
	FileName string

	// Repository is the globally unique name of the repo of the
	// match
	Repository  string
	Branches    []string
	LineMatches []LineMatch

	// Only set if requested
	Content []byte

	// SubRepositoryName is the globally unique name of the repo,
	// if it came from a subrepository
	SubRepositoryName string

	// SubRepositoryPath holds the prefix where the subrepository
	// was mounted.
	SubRepositoryPath string

	// Commit SHA1 (hex) of the (sub)repo holding the file.
	Version string
}

// LineMatch holds the matches within a single line in a file.
type LineMatch struct {
	// The line in which a match was found.
	Line       []byte
	LineStart  int
	LineEnd    int
	LineNumber int

	// If set, this was a match on the filename.
	FileName bool

	// The higher the better. Only ranks the quality of the match
	// within the file, does not take rank of file into account
	Score         float64
	LineFragments []LineFragmentMatch
}

// LineFragmentMatch a segment of matching text within a line.
type LineFragmentMatch struct {
	// Offset within the line.
	LineOffset int

	// Offset from file start
	Offset uint32

	// Number bytes that match.
	MatchLength int
}

// Stats contains interesting numbers on the search
type Stats struct {
	// Total length of files loaded.
	BytesLoaded int64

	// Number of search shards that had a crash.
	Crashes int

	// Wall clock time for this search
	Duration time.Duration

	// Number of files containing a match.
	FileCount int

	// Files that we evaluated. Equivalent to files for which all
	// atom matches (including negations) evaluated to true.
	FilesConsidered int

	// Files for which we loaded file content to verify substring matches
	FilesLoaded int

	// Candidate files whose contents weren't examined because we
	// gathered enough matches.
	FilesSkipped int

	// Number of non-overlapping matches
	MatchCount int

	// Number of candidate matches as a result of searching ngrams.
	NgramMatches int

	// Wall clock time for queued search.
	Wait time.Duration
}

func (s *Stats) Add(o Stats) {
	s.BytesLoaded += o.BytesLoaded
	s.Crashes += o.Crashes
	s.FileCount += o.FileCount
	s.FilesConsidered += o.FilesConsidered
	s.FilesLoaded += o.FilesLoaded
	s.FilesSkipped += o.FilesSkipped
	s.MatchCount += o.MatchCount
	s.NgramMatches += o.NgramMatches
}

// SearchResult contains search matches and extra data
type SearchResult struct {
	Stats
	Files []FileMatch

	// RepoURLs holds a repo => template string map.
	RepoURLs map[string]string

	// FragmentNames holds a repo => template string map, for
	// the line number fragment.
	LineFragments map[string]string
}

// RepositoryBranch describes an indexed branch, which is a name
// combined with a version.
type RepositoryBranch struct {
	Name    string
	Version string
}

// Repository holds repository metadata.
type Repository struct {
	// The repository name
	Name string
	// The repository URL.
	URL string

	// The branches indexed in this repo.
	Branches []RepositoryBranch

	// Nil if this is not the super project.
	SubRepoMap map[string]*Repository

	// URL template to link to the commit of a branch
	CommitURLTemplate string

	// The repository URL for getting to a file.  Has access to
	// {{Branch}}, {{Path}}
	FileURLTemplate string

	// The URL fragment to add to a file URL for line numbers.
	// has access to {{LineNumber}}.
	LineFragmentTemplate string
}

// IndexMetadata holds metadata stored in the index file.
type IndexMetadata struct {
	IndexFormatVersion  int
	IndexFeatureVersion int
	IndexTime           time.Time
}

// Statistics of a (collection of) repositories.
type RepoStats struct {
	// Repos is used for aggregrating the number of repositories.
	Repos int

	// Shards is the total number of search shards.
	Shards int

	// Documents holds the number of documents or files.
	Documents int

	// IndexBytes is the amount of RAM used for index overhead.
	IndexBytes int64

	// ContentBytes is the amount of RAM used for raw content.
	ContentBytes int64
}

func (s *RepoStats) Add(o *RepoStats) {
	// can't update Repos, since one repo may have multiple
	// shards.
	s.Shards += o.Shards
	s.IndexBytes += o.IndexBytes
	s.Documents += o.Documents
	s.ContentBytes += o.ContentBytes
}

type RepoListEntry struct {
	Repository    Repository
	IndexMetadata IndexMetadata
	Stats         RepoStats
}

// RepoList holds a set of Repository metadata.
type RepoList struct {
	Repos   []*RepoListEntry
	Crashes int
}

type Searcher interface {
	Search(ctx context.Context, q query.Q, opts *SearchOptions) (*SearchResult, error)

	// List lists repositories. The query `q` can only contain
	// query.Repo atoms.
	List(ctx context.Context, q query.Q) (*RepoList, error)
	Close()

	// Describe the searcher for debug messages.
	String() string
}

type SearchOptions struct {
	// Return the whole file.
	Whole bool

	// Maximum number of matches: skip all processing an index
	// shard after we found this many non-overlapping matches.
	ShardMaxMatchCount int

	// Maximum number of matches: stop looking for more matches
	// once we have this many matches across shards.
	TotalMaxMatchCount int

	// Maximum number of important matches: skip processing
	// shard after we found this many important matches.
	ShardMaxImportantMatch int

	// Maximum number of important matches across shards.
	TotalMaxImportantMatch int

	// Abort the search after this much time has passed.
	MaxWallTime time.Duration
}

func (s *SearchOptions) String() string {
	return fmt.Sprintf("%#v", s)
}
