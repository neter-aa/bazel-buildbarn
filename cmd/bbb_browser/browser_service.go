package main

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/EdSchouten/bazel-buildbarn/pkg/ac"
	"github.com/EdSchouten/bazel-buildbarn/pkg/blobstore"
	"github.com/EdSchouten/bazel-buildbarn/pkg/cas"
	"github.com/EdSchouten/bazel-buildbarn/pkg/util"
	remoteexecution "github.com/bazelbuild/remote-apis/build/bazel/remote/execution/v2"
	"github.com/golang/protobuf/proto"
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
	router.HandleFunc("/action/{instance}/{hash}/{sizeBytes}/", s.handleActionFromActionCache)
	router.HandleFunc("/action/{instance}/{hash}/{sizeBytes}/{actionResult}", s.handleActionFromURL)
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

func (s *BrowserService) handleActionFromActionCache(w http.ResponseWriter, req *http.Request) {
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

	s.handleAction(w, req, digest, actionResult)
}

func (s *BrowserService) handleActionFromURL(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data, err := base64.URLEncoding.DecodeString(mux.Vars(req)["actionResult"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var actionResult remoteexecution.ActionResult
	if err := proto.Unmarshal(data, &actionResult); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	s.handleAction(w, req, digest, &actionResult)
}

func (s *BrowserService) handleAction(w http.ResponseWriter, req *http.Request, digest *util.Digest, actionResult *remoteexecution.ActionResult) {
	actionInfo := struct {
		Instance     string
		Action       *remoteexecution.Action
		Command      *remoteexecution.Command
		InputRoot    *directoryInfo
		ActionResult *remoteexecution.ActionResult
	}{
		Instance:     digest.GetInstance(),
		ActionResult: actionResult,
	}

	ctx := req.Context()
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
