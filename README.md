# [free-gpt3.5-2api](https://github.com/aurorax-neo/free-gpt3.5-2api)

## 接口

#### /v1/chat/completions

###### 支持返回stream和json

```
http://<ip>:<port>/v1/chat/completions
```

##### 示例

```
curl http://127.0.0.1:9846
```

```
curl --location --request POST 'http://127.0.0.1:9846/v1/chat/completions' \
--header 'Authorization: Bearer abc' \
--header 'Content-Type: application/json' \
--data-raw '{
    "model": "gpt-3.5-turbo",
    "messages": [
        {
            "role": "user",
            "content": "西红柿炒钢丝球怎么做?"
        }
    ],
    "stream": false
}'
```

## 配置

### 环境变量

```
LOG_LEVEL=info   # debug, info, warn, error
BIND=0.0.0.0     # 127.0.0.1
PORT=3040
PROXY=			 # http://127.0.0.1:7890
AUTHORIZATIONS=  # abc,bac (英文 , 分隔)
POOL_MAX_COUNT=5 # max number of connections to keep in the pool
AUTH_ED=180      # expiration time for the authorization in seconds
AUTH_USE_COUNT=5 # number of times an authorization can be used
```

###### 也可使用与程序同目录下 `.env` 文件配置上述字段


### docker部署

##### 1 .创建文件夹

```
mkdir -p $PWD/free-gpt3.5-2api
```

##### 2.拉取镜像启动

###### 注：AUTH_TOKEN自行替换；tag替换为release版本号，如：0.0.1

```
docker run -itd  --name=free-gpt3.5-2api -p 9846:3040 -v $PWD/free-gpt3.5-2api/logs:/app/logs registry.cn-hangzhou.aliyuncs.com/aurorax/free-gpt3.5-2api:<tag>
```

