package service

import (
	"app/internal/provider"
	"app/pkg/common"
	pkgProvider "app/pkg/provider"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	rtctokenbuilder "github.com/AgoraIO/Tools/DynamicKey/AgoraDynamicKey/go/src/rtctokenbuilder2"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gogf/gf/container/gmap"
	"github.com/gogf/gf/crypto/gmd5"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	privilegeExpirationInSeconds = uint32(86400)
	tokenExpirationInSeconds     = uint32(86400)
)

var (
	logTag = slog.String("service", "MAIN_SERVICE")
)

type MainService struct {
	deps    MainServiceDepends
	workers *gmap.Map
}

type MainServiceDepends struct {
	Config           MainServiceConfig
	ManifestProvider *provider.ManifestProvider
}

type MainServiceConfig struct {
	AppId                    string
	AppCertificate           string
	TTSVendorChinese         string
	TTSVendorEnglish         string
	WorkersMax               int
	WorkerQuitTimeoutSeconds int
}

func NewMainService(deps MainServiceDepends) *MainService {
	return &MainService{
		deps:    deps,
		workers: gmap.New(true),
	}
}

func (s *MainService) output(c *gin.Context, code *common.Code, data any, httpStatus ...int) {
	if len(httpStatus) == 0 {
		httpStatus = append(httpStatus, http.StatusOK)
	}

	c.JSON(httpStatus[0], gin.H{"code": code.Code, "msg": code.Msg, "data": data})
}

func (s *MainService) HandlerHealth(c *gin.Context) {
	slog.Debug("handlerHealth", logTag)
	s.output(c, common.CodeOk, nil)
}

func (s *MainService) HandlerPing(c *gin.Context) {
	var req common.PingReq

	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		slog.Error("handlerPing params invalid", "err", err, logTag)
		s.output(c, common.CodeErrParamsInvalid, http.StatusBadRequest)
		return
	}

	slog.Info("handlerPing start", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)

	if strings.TrimSpace(req.ChannelName) == "" {
		slog.Error("handlerPing channel empty", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrChannelEmpty, http.StatusBadRequest)
		return
	}

	if !s.workers.Contains(req.ChannelName) {
		slog.Error("handlerPing channel not existed", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrChannelNotExisted, http.StatusBadRequest)
		return
	}

	// Update worker
	worker := s.workers.Get(req.ChannelName).(*Worker)
	worker.UpdateTs = time.Now().Unix()

	slog.Info("handlerPing end", "worker", worker, "requestId", req.RequestId, logTag)
	s.output(c, common.CodeSuccess, nil)
}

// HandlerStart is a handle for start worker.
func (s *MainService) HandlerStart(c *gin.Context) {
	workersRunning := s.workers.Size()

	slog.Info("handlerStart start", "workersRunning", workersRunning, logTag)

	var req common.StartReq
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		slog.Error("handlerStart params invalid", "err", err, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrParamsInvalid, http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.ChannelName) == "" {
		slog.Error("handlerStart channel empty", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrChannelEmpty, http.StatusBadRequest)
		return
	}

	if workersRunning >= s.deps.Config.WorkersMax {
		slog.Error("handlerStart workers exceed", "workersRunning", workersRunning, "WorkersMax", s.deps.Config.WorkersMax, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrWorkersLimit, http.StatusTooManyRequests)
		return
	}

	if s.workers.Contains(req.ChannelName) {
		slog.Error("handlerStart channel existed", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrChannelExisted, http.StatusBadRequest)
		return
	}

	manifestJsonFile, logFile, err := s.createWorkerManifest(&req)
	if err != nil {
		slog.Error("handlerStart create worker manifest", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrProcessManifestFailed, http.StatusInternalServerError)
		return
	}

	worker := newWorker(req.ChannelName, logFile, manifestJsonFile)
	worker.QuitTimeoutSeconds = s.deps.Config.WorkerQuitTimeoutSeconds
	if err := worker.start(&req); err != nil {
		slog.Error("handlerStart start worker failed", "err", err, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrStartWorkerFailed, http.StatusInternalServerError)
		return
	}
	s.workers.SetIfNotExist(req.ChannelName, worker)

	slog.Info("handlerStart end", "workersRunning", s.workers.Size(), "worker", worker, "requestId", req.RequestId, logTag)
	s.output(c, common.CodeSuccess, nil)
}

func (s *MainService) HandlerStop(c *gin.Context) {
	var req common.StopReq

	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		slog.Error("handlerStop params invalid", "err", err, logTag)
		s.output(c, common.CodeErrParamsInvalid, http.StatusBadRequest)
		return
	}

	slog.Info("handlerStop start", "req", req, logTag)

	if strings.TrimSpace(req.ChannelName) == "" {
		slog.Error("handlerStop channel empty", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrChannelEmpty, http.StatusBadRequest)
		return
	}

	if !s.workers.Contains(req.ChannelName) {
		slog.Error("handlerStop channel not existed", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrChannelNotExisted, http.StatusBadRequest)
		return
	}

	worker := s.workers.Get(req.ChannelName).(*Worker)
	if err := worker.stop(req.RequestId, req.ChannelName); err != nil {
		slog.Error("handlerStop kill app failed", "err", err, "worker", s.workers.Get(req.ChannelName), "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrStopAppFailed, http.StatusInternalServerError)
		return
	}
	s.workers.Remove(req.ChannelName)

	slog.Info("handlerStop end", "requestId", req.RequestId, logTag)
	s.output(c, common.CodeSuccess, nil)
}

func (s *MainService) HandlerGenerateToken(c *gin.Context) {
	var req common.GenerateTokenReq

	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		slog.Error("handlerGenerateToken params invalid", "err", err, logTag)
		s.output(c, common.CodeErrParamsInvalid, http.StatusBadRequest)
		return
	}

	slog.Info("handlerGenerateToken start", "req", req, logTag)

	if strings.TrimSpace(req.ChannelName) == "" {
		slog.Error("handlerGenerateToken channel empty", "channelName", req.ChannelName, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrChannelEmpty, http.StatusBadRequest)
		return
	}

	if s.deps.Config.AppCertificate == "" {
		s.output(c, common.CodeSuccess, map[string]any{"appId": s.deps.Config.AppId, "token": s.deps.Config.AppId, "channel_name": req.ChannelName, "uid": req.Uid})
		return
	}

	token, err := rtctokenbuilder.BuildTokenWithUid(s.deps.Config.AppId, s.deps.Config.AppCertificate, req.ChannelName, req.Uid, rtctokenbuilder.RolePublisher, tokenExpirationInSeconds, privilegeExpirationInSeconds)
	if err != nil {
		slog.Error("handlerGenerateToken generate token failed", "err", err, "requestId", req.RequestId, logTag)
		s.output(c, common.CodeErrGenerateTokenFailed, http.StatusBadRequest)
		return
	}

	slog.Info("handlerGenerateToken end", "requestId", req.RequestId, logTag)
	s.output(c, common.CodeSuccess, map[string]any{"appId": s.deps.Config.AppId, "token": token, "channel_name": req.ChannelName, "uid": req.Uid})
}

// createWorkerManifest create worker temporary Mainfest.
func (s *MainService) createWorkerManifest(req *common.StartReq) (manifestJsonFile string, logFile string, err error) {
	vendor := s.getTtsVendor(req.AgoraAsrLanguage)
	tts := pkgProvider.GetTts(vendor)
	if tts == nil {
		err = errors.New(fmt.Sprintf("unknow tts vendor", vendor))
		slog.Error("handlerStart generate token failed", "err", err, "requestId", req.RequestId, logTag)
		return "", "", err
	}

	manifestJson, ok := s.deps.ManifestProvider.GetManifestJson(vendor)
	if !ok {
		err = errors.New(fmt.Sprintf("unknow manifest vendor", vendor))
		slog.Error("handlerStart get manifest json failed", "err", err, "requestId", req.RequestId, logTag)
		return "", "", err
	}

	if s.deps.Config.AppId != "" {
		manifestJson, _ = sjson.Set(manifestJson, `predefined_graphs.0.nodes.#(name=="agora_rtc").property.app_id`, s.deps.Config.AppId)
	}
	appId := gjson.Get(manifestJson, `predefined_graphs.0.nodes.#(name=="agora_rtc").property.app_id`).String()

	// Generate token
	token := appId
	if s.deps.Config.AppCertificate != "" {
		token, err = rtctokenbuilder.BuildTokenWithUid(appId, s.deps.Config.AppCertificate, req.ChannelName, 0, rtctokenbuilder.RoleSubscriber, tokenExpirationInSeconds, privilegeExpirationInSeconds)
		if err != nil {
			slog.Error("handlerStart generate token failed", "err", err, "requestId", req.RequestId, logTag)
			return "", "", err
		}
	}

	manifestJson, _ = sjson.Set(manifestJson, `predefined_graphs.0.nodes.#(name=="agora_rtc").property.token`, token)
	if req.AgoraAsrLanguage != "" {
		manifestJson, _ = sjson.Set(manifestJson, `predefined_graphs.0.nodes.#(name=="agora_rtc").property.agora_asr_language`, req.AgoraAsrLanguage)
	}
	if req.ChannelName != "" {
		manifestJson, _ = sjson.Set(manifestJson, `predefined_graphs.0.nodes.#(name=="agora_rtc").property.channel`, req.ChannelName)
	}
	if req.RemoteStreamId != 0 {
		manifestJson, _ = sjson.Set(manifestJson, `predefined_graphs.0.nodes.#(name=="agora_rtc").property.remote_stream_id`, req.RemoteStreamId)
	}

	language := gjson.Get(manifestJson, `predefined_graphs.0.nodes.#(name=="agora_rtc").property.agora_asr_language`).String()
	manifestJson, err = tts.ProcessManifest(manifestJson, common.Language(language), req.VoiceType)
	if err != nil {
		slog.Error("handlerStart tts ProcessManifest failed", "err", err, "requestId", req.RequestId, logTag)
		return "", "", err
	}

	channelNameMd5 := gmd5.MustEncryptString(req.ChannelName)
	ts := time.Now().UnixNano()
	manifestJsonFile = fmt.Sprintf("/tmp/manifest-%s-%d.json", channelNameMd5, ts)
	logFile = fmt.Sprintf("/tmp/app-%s-%d.log", channelNameMd5, ts)
	err = os.WriteFile(manifestJsonFile, []byte(manifestJson), 0644)
	if err != nil {
		slog.Error("handlerStart write manifest.json failed", "err", err, "manifestJsonFile", manifestJsonFile, "requestId", req.RequestId, logTag)
		return "", "", err
	}

	return manifestJsonFile, logFile, nil
}

// CleanWorker clean unused workers in background.
func (s *MainService) CleanWorker() {
	for {
		for _, channelName := range s.workers.Keys() {
			worker := s.workers.Get(channelName).(*Worker)

			nowTs := time.Now().Unix()
			if worker.UpdateTs+int64(worker.QuitTimeoutSeconds) < nowTs {
				if err := worker.stop(uuid.New().String(), channelName.(string)); err != nil {
					slog.Error("Worker cleanWorker failed", "err", err, "channelName", channelName, logTag)
					continue
				}

				slog.Info("Worker cleanWorker success", "channelName", channelName, "worker", worker, "nowTs", nowTs, logTag)
			}
		}

		slog.Debug("Worker cleanWorker sleep", "sleep", workerCleanSleepSeconds, logTag)
		time.Sleep(workerCleanSleepSeconds * time.Second)
	}
}

func (s *MainService) getTtsVendor(language common.Language) string {
	if language == common.LanguageChinese {
		return s.deps.Config.TTSVendorChinese
	}

	return s.deps.Config.TTSVendorEnglish
}
