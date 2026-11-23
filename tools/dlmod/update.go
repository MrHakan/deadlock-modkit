package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

// dlmod only goes online for "dlmod update", browsing mods and the character creator. "dlmod
// update" downloads the Counter Intelligence HUD built for the installed game build from the
// project's GitLab releases. Everything else works offline.
//
// Downloads go through the system's curl, which Windows 10 and later include and Linux systems
// have, rather than Go's HTTP and TLS stack, which would double the size of dlmod. curl also
// brings proxy settings, redirects and the system's certificate store.

// projectURL is the project's home page, for messages.
const projectURL = "https://gitlab.com/mrhakan/deadlock-modkit"

// releasesURL lists the project's releases, newest first. It names the project by its number,
// which survives a rename and avoids an escaped slash in the path. Tests point it at a local
// server.
var releasesURL = "https://gitlab.com/api/v4/projects/89335829/releases?per_page=100"

var hudAssetName = regexp.MustCompile(`^deadlock-counter-intelligence-(\d+)-hud-en\.zip$`)

// releaseLink is one file attached to a GitLab release. direct_asset_url is the release's
// permanent address for it, which redirects to the file.
type releaseLink struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	DirectURL string `json:"direct_asset_url"`
}

// address prefers the release's own address for the file, and accepts only HTTPS.
func (l releaseLink) address() (string, error) {
	for _, candidate := range []string{l.DirectURL, l.URL} {
		if candidate != "" && allowedURL(candidate) == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("the release lists %s without an HTTPS address", l.Name)
}

type release struct {
	Tag        string `json:"tag_name"`
	Upcoming   bool   `json:"upcoming_release"`
	ReleasedAt string `json:"released_at"`
	Assets     struct {
		Links []releaseLink `json:"links"`
	} `json:"assets"`
}

// published leaves out releases dated in the future and pre-releases (a tag such as v0.4.1-beta).
func (r release) published() bool {
	return !r.Upcoming && !strings.Contains(r.Tag, "-")
}

func (r release) link(name string) (releaseLink, bool) {
	for _, link := range r.Assets.Links {
		if link.Name == name {
			return link, true
		}
	}
	return releaseLink{}, false
}

// allowedURL accepts HTTPS addresses, and plain HTTP only for this computer (tests).
func allowedURL(address string) error {
	parsed, err := url.Parse(address)
	if err != nil {
		return err
	}
	if parsed.Scheme == "https" || localHTTP(parsed) {
		return nil
	}
	return fmt.Errorf("refusing to download from %s", address)
}

func localHTTP(parsed *url.URL) bool {
	host := parsed.Hostname()
	return parsed.Scheme == "http" && (host == "127.0.0.1" || host == "localhost")
}

// statusError is a download the server answered with an HTTP error.
type statusError struct {
	Status int
	Detail string
}

func (e *statusError) Error() string { return "download failed: " + e.Detail }

var curlStatus = regexp.MustCompile(`returned error: (\d{3})`)

// fetch streams an address into w through curl. Redirects are followed over HTTPS only (and
// over HTTP between local test servers).
func fetch(address string, w io.Writer) error {
	if err := allowedURL(address); err != nil {
		return err
	}
	protocols := "=https"
	if parsed, _ := url.Parse(address); localHTTP(parsed) {
		protocols = "=http,https"
	}
	command := exec.Command("curl", "--fail", "--silent", "--show-error", "--location",
		"--proto", protocols, "--proto-redir", protocols, "--max-time", "300",
		"--user-agent", "dlmod/"+version, "--url", address)
	var problems bytes.Buffer
	command.Stdout, command.Stderr = w, &problems
	if err := command.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.New("curl was not found; it comes with Windows 10 and later. Download the HUD from " +
				projectURL + "/-/releases instead and drag it onto dlmod")
		}
		detail := strings.TrimSpace(problems.String())
		if match := curlStatus.FindStringSubmatch(detail); match != nil {
			status, _ := strconv.Atoi(match[1])
			return &statusError{Status: status, Detail: detail}
		}
		return fmt.Errorf("download failed: %s", detail)
	}
	return nil
}

// cappedWriter keeps at most limit bytes and fails past them, for small files such as checksums.
type cappedWriter struct {
	bytes.Buffer
	limit int
}

func (w *cappedWriter) Write(data []byte) (int, error) {
	if w.Len()+len(data) > w.limit {
		return 0, fmt.Errorf("the file is larger than %d bytes", w.limit)
	}
	return w.Buffer.Write(data)
}

var sha256Hex = regexp.MustCompile(`(?i)\b[0-9a-f]{64}\b`)

// parseChecksum reads the SHA-256 of name from a checksum file as people write them: "<hex>",
// sha256sum's "<hex>  <name>", or PowerShell's Get-FileHash table, in UTF-8 or UTF-16, with or
// without a byte order mark, upper or lower case. A file with several hashes must name the file
// on the hash's line.
func parseChecksum(data []byte, name string) (string, error) {
	text := decodeText(data)
	var found []string
	for _, line := range strings.Split(text, "\n") {
		hashes := sha256Hex.FindAllString(line, -1)
		if len(hashes) == 1 && strings.Contains(line, name) {
			return strings.ToLower(hashes[0]), nil
		}
		for _, hash := range hashes {
			found = append(found, strings.ToLower(hash))
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	if len(found) == 0 {
		return "", errors.New("it has no SHA-256")
	}
	return "", fmt.Errorf("it has %d hashes and none is on a line naming %s", len(found), name)
}

// decodeText turns UTF-16 (as PowerShell 5.1 writes with ">") or UTF-8 into a string.
func decodeText(data []byte) string {
	littleEndian := len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE
	bigEndian := len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF
	if !littleEndian && !bigEndian && len(data) >= 2 && data[1] == 0 && data[0] != 0 {
		littleEndian = true // UTF-16 without a byte order mark
	}
	if littleEndian || bigEndian {
		if data[0] == 0xFF || data[0] == 0xFE {
			data = data[2:]
		}
		units := make([]uint16, len(data)/2)
		for i := range units {
			if littleEndian {
				units[i] = uint16(data[2*i]) | uint16(data[2*i+1])<<8
			} else {
				units[i] = uint16(data[2*i])<<8 | uint16(data[2*i+1])
			}
		}
		return string(utf16.Decode(units))
	}
	return strings.TrimPrefix(string(data), string(rune(0xFEFF)))
}

// newestFirst orders releases by their release date, newest first, as GitLab lists them.
func newestFirst(releases []release) {
	stamp := func(r release) time.Time {
		parsed, _ := time.Parse(time.RFC3339, r.ReleasedAt)
		return parsed
	}
	sort.SliceStable(releases, func(i, j int) bool { return stamp(releases[i]).After(stamp(releases[j])) })
}

// hudRelease finds the newest published release with a HUD for the game build, and otherwise
// reports the newest build any release covers.
func hudRelease(releases []release, build int) (release, releaseLink, int, string) {
	newestFirst(releases)
	newest, newestTag := 0, ""
	for _, candidate := range releases {
		if !candidate.published() {
			continue
		}
		for _, link := range candidate.Assets.Links {
			match := hudAssetName.FindStringSubmatch(link.Name)
			if match == nil {
				continue
			}
			assetBuild, _ := strconv.Atoi(match[1])
			if assetBuild == build {
				return candidate, link, assetBuild, candidate.Tag
			}
			if assetBuild > newest {
				newest, newestTag = assetBuild, candidate.Tag
			}
		}
	}
	return release{}, releaseLink{}, newest, newestTag
}

// downloadFile writes a file to disk and returns its SHA-256.
func downloadFile(address, target string, total int64, label string, p prompter) (string, error) {
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	sinks := []io.Writer{file, hash}
	if total > 0 {
		sinks = append(sinks, &progressWriter{label: label, total: total, report: p})
	}
	err = fetch(address, io.MultiWriter(sinks...))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return hex.EncodeToString(hash.Sum(nil)), err
}

// releaseChecksum downloads the "<file>.sha256" published next to a release file. A release
// without one is refused: the checksum is what tells a complete download from a broken one.
func releaseChecksum(found release, name string) (string, error) {
	link, ok := found.link(name + ".sha256")
	if !ok {
		return "", fmt.Errorf("release %s has no %s.sha256 to check the download against; nothing was downloaded", found.Tag, name)
	}
	address, err := link.address()
	if err != nil {
		return "", err
	}
	body := &cappedWriter{limit: 4096}
	if err := fetch(address, body); err != nil {
		return "", err
	}
	sum, err := parseChecksum(body.Bytes(), name)
	if err != nil {
		return "", fmt.Errorf("%s.sha256 of release %s cannot be read: %v", name, found.Tag, err)
	}
	return sum, nil
}

// updateHUD installs the HUD release for the installed game build, replacing an older version
// in its slot. Installing the same version again changes nothing.
func (s *Store) updateHUD(p prompter) error {
	build := s.game.Version()
	if build == 0 {
		return errors.New("the game build is unknown because game/citadel/steam.inf is missing")
	}
	var listing bytes.Buffer
	if err := fetch(releasesURL, &listing); err != nil {
		var status *statusError
		if errors.As(err, &status) && status.Status == 404 {
			return fmt.Errorf("GitLab did not find the project's releases (%s). The project or its releases may not be public", projectURL)
		}
		if errors.As(err, &status) && status.Status == 429 {
			return errors.New("GitLab is limiting requests right now; try again in a few minutes")
		}
		return err
	}
	var releases []release
	if err := json.Unmarshal(listing.Bytes(), &releases); err != nil {
		return fmt.Errorf("unexpected answer from GitLab: %v", err)
	}
	found, asset, newest, newestTag := hudRelease(releases, build)
	if asset.Name == "" {
		if newest > 0 {
			return fmt.Errorf("there is no HUD for game build %d yet; the newest release (%s) is for build %d. "+
				"A new one usually follows a game update within a day", build, newestTag, newest)
		}
		return fmt.Errorf("no release has a HUD for game build %d", build)
	}
	want, err := releaseChecksum(found, asset.Name)
	if err != nil {
		return err
	}
	address, err := asset.address()
	if err != nil {
		return err
	}
	folder := filepath.Join(s.game.StateDir(), "downloads")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		return err
	}
	target := filepath.Join(folder, asset.Name)
	defer os.Remove(target)
	sum, err := downloadFile(address, target, 0, asset.Name, p)
	if err != nil {
		return err
	}
	if sum != want {
		return errors.New("the download does not match the checksum published with the release; nothing was installed")
	}
	p.say("Downloaded the HUD for build %d from release %s.", build, found.Tag)
	return s.install(p, target, "Deadlock Counter Intelligence "+found.Tag)
}
