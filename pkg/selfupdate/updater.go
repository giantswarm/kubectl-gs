package selfupdate

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/giantswarm/microerror"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

type Config struct {
	// GithubToken will be used when fetching versions
	// from private GitHub repositories.
	GithubToken string
	// CurrentVersion is the currently installed version
	// of the application.
	CurrentVersion string
	// RepositoryURL is the URL to the GitHub repository.
	RepositoryURL string
	// CacheDir is the path to the directory where the
	// cache should be stored.
	CacheDir string
}

type Updater struct {
	githubToken    string
	currentVersion semver.Version
	repository     string
	cacheDir       string

	selfUpdater *selfupdate.Updater
	cache       *cache
}

func New(c Config) (*Updater, error) {
	if len(c.CurrentVersion) < 1 {
		return nil, microerror.Maskf(invalidConfigError, "%T.CurrentVersion must not be empty", c)
	}
	if len(c.RepositoryURL) < 1 {
		return nil, microerror.Maskf(invalidConfigError, "%T.RepositoryURL must not be empty", c)
	}

	var err error

	u := &Updater{
		githubToken: c.GithubToken,
		cacheDir:    c.CacheDir,
	}

	{
		u.repository, err = u.parseRepoFromURL(c.RepositoryURL)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		u.currentVersion, err = semver.Parse(c.CurrentVersion)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		u.selfUpdater, err = selfupdate.NewUpdater(selfupdate.Config{
			APIToken: u.githubToken,
		})
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	{
		u.cache = &cache{}
		if len(u.cacheDir) > 0 {
			// We ignore the error on purpose, since we'll just override the config file
			// if it's invalid.
			_ = u.cache.Restore(u.cacheDir)
		}
	}

	return u, nil
}

// InstallLatest installs the newest version that can
// be installed.
func (u *Updater) InstallLatest() error {
	_, err := u.selfUpdater.UpdateSelf(u.currentVersion, u.repository)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// GetLatest returns the latest version available in the
// source repository, and if we can upgrade to that version
// or not (it can be equal to the current version).
func (u *Updater) GetLatest() (version string, err error) {
	latestVersion, err := u.getLatestVersion()
	if err != nil {
		return "", microerror.Mask(err)
	}

	if latestVersion.GT(u.currentVersion) {
		err = microerror.Maskf(hasNewVersionError, "Version %s available", latestVersion.String())
	} else {
		err = nil
	}

	return latestVersion.String(), err
}

func (u *Updater) getLatestVersion() (semver.Version, error) {
	allowCache := len(u.cacheDir) > 0

	if allowCache && !u.cache.IsExpired() {
		version, err := semver.Parse(u.cache.LatestVersion)
		// If this not a valid semver version, then it means
		// that the someone fiddled with the cache file. We'll
		// just override it.
		if err == nil {
			return version, nil
		}
	}

	latestVersion, _, err := u.selfUpdater.DetectLatest(u.repository)
	if err != nil {
		return semver.Version{}, microerror.Mask(err)
	}

	if latestVersion == nil {
		return semver.Version{}, microerror.Maskf(versionNotFoundError, "couldn't find the latest version and/or release assets on GitHub, probably due to token without access to the repository %s.", u.repository)
	}

	if allowCache {
		u.cache.LatestVersion = latestVersion.Version.String()

		err = u.cache.Persist(u.cacheDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Failed to persist the cache with error: %s", err)
		}
	}

	return latestVersion.Version, nil
}

func (u *Updater) parseRepoFromURL(sourceURL string) (string, error) {
	repo, err := url.Parse(sourceURL)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return strings.TrimPrefix(repo.Path, "/"), nil
}
