package chat

import (
	"encoding/json"
	"fmt"
	"free-gpt3.5-2api/chat"
	"free-gpt3.5-2api/common"
	"free-gpt3.5-2api/pool"
	v1 "free-gpt3.5-2api/v1"
	"free-gpt3.5-2api/v1/chat/reqmodel"
	"free-gpt3.5-2api/v1/chat/respmodel"
	"github.com/aurorax-neo/go-logger"
	fhttp "github.com/bogdanfinn/fhttp"
	"github.com/gin-gonic/gin"
	"github.com/launchdarkly/eventsource"
	"io"
	"net/http"
	"strings"
	"time"
)

func gpt35(c *gin.Context, apiReq *reqmodel.ApiReq) {
	// 获取 chat 实例
	instance := pool.GetGpt35PoolInstance().GetGpt35(3)
	if instance == nil {
		v1.ErrorResponse(c, http.StatusInternalServerError, "instance is nil", nil)
		logger.Logger.Error("instance is nil")
		pool.GetGpt35PoolInstance().RAGpt35AtIndex(pool.GetGpt35PoolInstance().Index)
		return
	}
	logger.Logger.Info(fmt.Sprint("Gpt35  index: ", pool.GetGpt35PoolInstance().Index))
	// 转换请求
	ChatReq35 := reqmodel.ApiReq2ChatReq35(apiReq)
	// 请求参数
	body, err := common.Struct2BytesBuffer(ChatReq35)
	if err != nil {
		v1.ErrorResponse(c, http.StatusInternalServerError, "", err)
		logger.Logger.Error(err.Error())
		return

	}
	// 生成请求
	request, err := instance.NewRequest(fhttp.MethodPost, chat.ApiUrl, body)
	// 设置请求头
	request.Header.Set("oai-device-id", instance.Session.OaiDeviceId)
	request.Header.Set("openai-sentinel-chat-requirements-token", instance.Session.Token)
	// 发送请求
	response, err := instance.Client.Do(request)
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body)
	if err != nil {
		v1.ErrorResponse(c, http.StatusInternalServerError, "", err)
		logger.Logger.Error(err.Error())
		return
	}
	if response.StatusCode != http.StatusOK {
		v1.ErrorResponse(c, response.StatusCode, "", nil)
		logger.Logger.Error(fmt.Sprint(response.StatusCode))
		pool.GetGpt35PoolInstance().RAGpt35AtIndex(pool.GetGpt35PoolInstance().Index)
		return
	}
	// 流式返回
	if apiReq.Stream {
		__CompletionsStream(c, apiReq, response)
	} else { // 非流式回应
		__CompletionsNoStream(c, apiReq, response)
	}
}

func __CompletionsStream(c *gin.Context, apiReq *reqmodel.ApiReq, resp *fhttp.Response) {
	messageTemp := ""
	decoder := eventsource.NewDecoder(resp.Body)
	defer func(decoder *eventsource.Decoder) {
		_, _ = decoder.Decode()
	}(decoder)
	// 响应id
	id := v1.GenerateID(29)
	handlingSigns := false
	for {
		event, err := decoder.Decode()
		if err != nil {
			v1.ErrorResponse(c, http.StatusInternalServerError, "", err)
			logger.Logger.Error(err.Error())
			break
		}
		name := event.Event()
		data := event.Data()
		// 空白数据不处理
		if data == "" {
			continue
		}
		// 结束标志
		if data == "[DONE]" {
			apiRespObj := &respmodel.StreamObj{}
			// id
			apiRespObj.ID = id
			// created
			apiRespObj.Created = time.Now().Unix()
			// object
			apiRespObj.Object = "chat.completion.chunk"
			// choices
			delta := respmodel.StreamDeltaObj{
				Content: "",
			}
			choices := respmodel.StreamChoiceObj{
				Delta:        delta,
				FinishReason: "stop",
			}
			apiRespObj.Choices = append(apiRespObj.Choices, choices)
			// model
			apiRespObj.Model = apiReq.Model
			// 生成响应
			bytes, err := common.Struct2Bytes(apiRespObj)
			if err != nil {
				logger.Logger.Error(err.Error())
				continue
			}
			// 发送响应
			c.SSEvent(name, fmt.Sprint(" ", string(bytes)))
			// 结束
			c.SSEvent(name, " [DONE]")
			break
		}
		chatResp35 := &respmodel.ChatResp35{}
		err = json.Unmarshal([]byte(data), chatResp35)
		// 脏数据不处理
		if err != nil {
			continue
		}
		// 仅处理assistant的消息
		if chatResp35.Message.Author.Role == "assistant" && (chatResp35.Message.Status == "in_progress" || handlingSigns) {
			// handlingSigns 置为 true
			handlingSigns = true
			// 仅处理第一个part
			parts := chatResp35.Message.Content.Parts[0]
			// 去除重复数据
			content := strings.Replace(parts, messageTemp, "", 1)
			messageTemp = parts
			// 空白数据不处理
			if content == "" {
				continue
			}
			// 生成响应 model
			apiRespObj := &respmodel.StreamObj{}
			// id
			apiRespObj.ID = id
			// created
			apiRespObj.Created = time.Now().Unix()
			// object
			apiRespObj.Object = "chat.completion.chunk"
			// choices
			delta := respmodel.StreamDeltaObj{
				Content: content,
			}
			choices := respmodel.StreamChoiceObj{
				Delta: delta,
			}
			apiRespObj.Choices = append(apiRespObj.Choices, choices)
			// model
			apiRespObj.Model = apiReq.Model
			// 生成响应
			bytes, err := common.Struct2Bytes(apiRespObj)
			if err != nil {
				logger.Logger.Error(err.Error())
				continue
			}
			// 发送响应
			c.SSEvent(name, fmt.Sprint(" ", string(bytes)))
			// 继续
			continue
		}
	}
}

func __CompletionsNoStream(c *gin.Context, apiReq *reqmodel.ApiReq, resp *fhttp.Response) {
	content := ""
	decoder := eventsource.NewDecoder(resp.Body)
	defer func(decoder *eventsource.Decoder) {
		_, _ = decoder.Decode()
	}(decoder)
	handlingSigns := false
	for {
		event, err := decoder.Decode()
		if err != nil {
			v1.ErrorResponse(c, http.StatusInternalServerError, "", err)
			logger.Logger.Error(err.Error())
			break
		}
		data := event.Data()
		// 空白数据不处理
		if data == "" {
			continue
		}
		// 结束标志
		if data == "[DONE]" {
			apiRespObj := &respmodel.JsonObj{}
			// id
			apiRespObj.ID = v1.GenerateID(29)
			// created
			apiRespObj.Created = time.Now().Unix()
			// object
			apiRespObj.Object = "chat.completion"
			// model
			apiRespObj.Model = apiReq.Model
			// usage
			usage := respmodel.JsonUsageObj{
				PromptTokens:     0,
				CompletionTokens: 0,
				TotalTokens:      0,
			}
			apiRespObj.Usage = usage
			// choices
			message := respmodel.JsonMessageObj{
				Role:    "assistant",
				Content: content,
			}
			choice := respmodel.JsonChoiceObj{
				Message:      message,
				FinishReason: "stop",
				Index:        0,
			}
			apiRespObj.Choices = append(apiRespObj.Choices, choice)
			// 返回响应
			c.JSON(http.StatusOK, apiRespObj)
			break
		}
		chatResp35 := &respmodel.ChatResp35{}
		err = json.Unmarshal([]byte(data), chatResp35)
		// 脏数据不处理
		if err != nil {
			continue
		}
		// 仅处理assistant的消息
		if chatResp35.Message.Author.Role == "assistant" && (chatResp35.Message.Status == "in_progress" || handlingSigns) {
			// handlingSigns 置为 true
			handlingSigns = true
			// 如果不包含上一次的数据则不处理
			if !strings.Contains(chatResp35.Message.Content.Parts[0], content) {
				continue
			}
			// 仅处理第一个part
			content = chatResp35.Message.Content.Parts[0]
			// 空白数据不处理
			if content == "" {
				continue
			}
			continue
		}
	}
}
