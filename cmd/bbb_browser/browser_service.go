package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/buildkite/terminal"
	"github.com/gorilla/mux"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func getDigestFromRequest(req *http.Request) (*util.Digest, error) {
	vars := mux.Vars(req)
	sizeBytes, err := strconv.ParseInt(vars["sizeBytes"], 10, 64)
	if err != nil {
		return nil, err
	}
	return util.NewDigest(
		vars["instance"],
		&remoteexecution.Digest{
			Hash:      vars["hash"],
			SizeBytes: sizeBytes,
		})
}

type BrowserService struct {
	contentAddressableStorage           cas.ContentAddressableStorage
	contentAddressableStorageBlobAccess blobstore.BlobAccess
	actionCache                         ac.ActionCache
	templates                           *template.Template
}

func NewBrowserService(contentAddressableStorage cas.ContentAddressableStorage, contentAddressableStorageBlobAccess blobstore.BlobAccess, actionCache ac.ActionCache, templates *template.Template, router *mux.Router) *BrowserService {
	s := &BrowserService{
		contentAddressableStorage:           contentAddressableStorage,
		contentAddressableStorageBlobAccess: contentAddressableStorageBlobAccess,
		actionCache:                         actionCache,
		templates:                           templates,
	}
	router.HandleFunc("/action/{instance}/{hash}/{sizeBytes}/", s.handleAction)
	router.HandleFunc("/actionfailure/{instance}/{hash}/{sizeBytes}/", s.handleActionFailure)
	router.HandleFunc("/command/{instance}/{hash}/{sizeBytes}/", s.handleCommand)
	router.HandleFunc("/directory/{instance}/{hash}/{sizeBytes}/", s.handleDirectory)
	router.HandleFunc("/file/{instance}/{hash}/{sizeBytes}/{{name}}", s.handleFile)
	router.HandleFunc("/tree/{instance}/{hash}/{sizeBytes}/", s.handleTree)
	return s
}

type directoryInfo struct {
	Instance  string
	Directory *remoteexecution.Directory
}

type logInfo struct {
	Name     string
	Instance string
	Digest   *remoteexecution.Digest
	TooLarge bool
	NotFound bool
	HTML     template.HTML
}

func (s *BrowserService) handleAction(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	actionResult, err := s.actionCache.GetActionResult(ctx, digest)
	if err != nil && status.Code(err) != codes.NotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.handleActionCommon(w, req, digest, actionResult)
}

func (s *BrowserService) handleActionFailure(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := req.Context()
	actionFailure, err := s.contentAddressableStorage.GetActionFailure(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	actionDigest, err := digest.NewDerivedDigest(actionFailure.ActionDigest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.handleActionCommon(w, req, actionDigest, actionFailure.ActionResult)
}

func (s *BrowserService) getLogInfo(ctx context.Context, name string, instance string, logDigest *remoteexecution.Digest) (*logInfo, error) {
	if logDigest == nil {
		return nil, nil
	}
	digest, err := util.NewDigest(instance, logDigest)
	if err != nil {
		return nil, err
	}
	if size := digest.GetSizeBytes(); size == 0 {
		// No log file present.
		return nil, nil
	} else if size > 100000 {
		// Log file too large to show inline.
		return &logInfo{
			Name:     name,
			Instance: instance,
			Digest:   logDigest,
			TooLarge: true,
		}, nil
	}

	r := s.contentAddressableStorageBlobAccess.Get(ctx, digest)
	data, err := ioutil.ReadAll(r)
	r.Close()
	if err == nil {
		// Log found. Convert ANSI escape sequences to HTML.
		return &logInfo{
			Name:     name,
			Instance: instance,
			Digest:   logDigest,
			HTML:     template.HTML(terminal.Render(data)),
		}, nil
	} else if status.Code(err) == codes.NotFound {
		// Not found.
		return &logInfo{
			Name:     name,
			Instance: instance,
			Digest:   logDigest,
			NotFound: true,
		}, nil
	} else {
		return nil, err
	}
}

func (s *BrowserService) handleActionCommon(w http.ResponseWriter, req *http.Request, digest *util.Digest, actionResult *remoteexecution.ActionResult) {
	instance := digest.GetInstance()
	actionInfo := struct {
		Instance string
		Action   *remoteexecution.Action

		Command *remoteexecution.Command

		ActionResult *remoteexecution.ActionResult
		StdoutInfo   *logInfo
		StderrInfo   *logInfo

		InputRoot *directoryInfo

		OutputDirectories  []*remoteexecution.OutputDirectory
		OutputSymlinks     []*remoteexecution.OutputSymlink
		OutputFiles        []*remoteexecution.OutputFile
		MissingDirectories []string
		MissingFiles       []string
	}{
		Instance:     instance,
		ActionResult: actionResult,
	}

	ctx := req.Context()
	if actionResult != nil {
		actionInfo.OutputDirectories = actionResult.OutputDirectories
		actionInfo.OutputSymlinks = actionResult.OutputFileSymlinks
		actionInfo.OutputFiles = actionResult.OutputFiles

		// TODO(edsch): Should we support Std{out,err}Raw as well? Buildbarn doesn't generate them.
		var err error
		actionInfo.StdoutInfo, err = s.getLogInfo(ctx, "Standard output", instance, actionResult.StdoutDigest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		actionInfo.StderrInfo, err = s.getLogInfo(ctx, "Standard error", instance, actionResult.StderrDigest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	action, err := s.contentAddressableStorage.GetAction(ctx, digest)
	if err == nil {
		actionInfo.Action = action

		commandDigest, err := digest.NewDerivedDigest(action.CommandDigest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		command, err := s.contentAddressableStorage.GetCommand(ctx, commandDigest)
		if err == nil {
			actionInfo.Command = command

			foundDirectories := map[string]bool{}
			for _, outputDirectory := range actionInfo.OutputDirectories {
				foundDirectories[outputDirectory.Path] = true
			}
			for _, outputDirectory := range command.OutputDirectories {
				if _, ok := foundDirectories[outputDirectory]; !ok {
					actionInfo.MissingDirectories = append(actionInfo.MissingDirectories, outputDirectory)
				}
			}
			foundFiles := map[string]bool{}
			for _, outputSymlinks := range actionInfo.OutputSymlinks {
				foundFiles[outputSymlinks.Path] = true
			}
			for _, outputFiles := range actionInfo.OutputFiles {
				foundFiles[outputFiles.Path] = true
			}
			for _, outputFile := range command.OutputFiles {
				if _, ok := foundFiles[outputFile]; !ok {
					actionInfo.MissingFiles = append(actionInfo.MissingFiles, outputFile)
				}
			}
		} else if status.Code(err) != codes.NotFound {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		inputRootDigest, err := digest.NewDerivedDigest(action.InputRootDigest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		directory, err := s.contentAddressableStorage.GetDirectory(ctx, inputRootDigest)
		if err == nil {
			actionInfo.InputRoot = &directoryInfo{
				Instance:  inputRootDigest.GetInstance(),
				Directory: directory,
			}
		} else if status.Code(err) != codes.NotFound {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else if status.Code(err) != codes.NotFound {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if action == nil && actionResult == nil {
		http.Error(w, "Could not find an action or action result", http.StatusNotFound)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "page_action.html", actionInfo); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) handleCommand(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	command, err := s.contentAddressableStorage.GetCommand(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "page_command.html", command); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) handleDirectory(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	directory, err := s.contentAddressableStorage.GetDirectory(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "page_directory.html", directoryInfo{
		Instance:  digest.GetInstance(),
		Directory: directory,
	}); err != nil {
		log.Print(err)
	}
}

func (s *BrowserService) handleFile(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	r := s.contentAddressableStorageBlobAccess.Get(ctx, digest)
	defer r.Close()

	// Attempt to read the first chunk of data to see whether we can
	// trigger an error. Only when no error occurs, we start setting
	// response headers.
	var first [4096]byte
	n, err := r.Read(first[:])
	if err != nil && err != io.EOF {
		// TODO(edsch): Convert error code.
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+mux.Vars(req)["name"])
	w.Header().Set("Content-Length", strconv.FormatInt(digest.GetSizeBytes(), 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(first[:n])
	io.Copy(w, r)
}

func (s *BrowserService) handleTree(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx := req.Context()
	tree, err := s.contentAddressableStorage.GetTree(ctx, digest)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Error(w, fmt.Sprintf("%s", tree), http.StatusBadRequest)
}
