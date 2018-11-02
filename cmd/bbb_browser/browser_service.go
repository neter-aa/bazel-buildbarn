package main

import (
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
	"github.com/gorilla/mux"
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

func (s *BrowserService) handleAction(w http.ResponseWriter, req *http.Request) {
	digest, err := getDigestFromRequest(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	actionInfo := struct {
		Instance  string
		Action    *remoteexecution.Action
		Command   *remoteexecution.Command
		InputRoot *directoryInfo
	}{
		Instance: digest.GetInstance(),
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
		} else {
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
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
