# CHAT 
## request
```json
    {
        "biz_info": {
            "sid": "2222",
            "cmd": "CHAT",
            "params": "123456"
        },
        "reqid":123456,
        "content": "简要介绍南京"
    }
```
**reqid 与 params的第一个参数**
## response
```json
    {
        "userid": "15314684226",
        "reqid": 123456,
        "event": {
            "sid": "",
            "event_type": "",
            "event_content": "",
            "event_params": ""
        },
        "content": "南京是中国江苏省省会，是中国历史文化名城之一。",
        "file_name": ""
    }
```

# ASR 先完成文件上传，然后再发生下面的指令
## request
```json
    {
        "biz_info": {
            "sid": "98765",
            "cmd": "ASR",
            "params": "654321"
        },
        "reqid":654321,
        "file_name": "recording-a974b925-b22f-4484-81ba-a76fc19169a9.m4a"

    }
```
## response
 1. 先返回ASR结果
```json
    {
        "userid": "15314684226",
        "reqid": 654321,
        "event": {
            "sid": "98765",
            "event_type": "ASR",
            "event_content": "",
            "event_params": "654321"
        },
        "content": ". .",
        "file_name": ""
    }
```
2. 返回TTS_STREAM
```json
   {
        "userid": "15314684226",
        "reqid": 9564,
        "event": {
            "sid": "98765",
            "event_type": "TTS_STREAM",
            "event_content": "Hello! How can I assist you today?\n",
            "event_params": "9564|-1|0"
        },
        "content": "",
        "file_name": "9564_-1_0"
    }
```


# IMAGE
## request 
```json
    {
        "biz_info": {
            "sid": "888",
            "cmd": "IMAGE",
            "params": "6543|ai.png|简单描述内容|true"
        },
        "reqid":6543,
        "file_name": "ai.png"

    }
```
## response
```json
    {
        "userid": "15314684226",
        "reqid": 6543,
        "event": {
            "sid": "888",
            "event_type": "IMAGE",
            "event_content": "",
            "event_params": "6543|ai.png"
        },
        "content": "图片中有一个标题，写着\"AI Tools for Education\"（教育用AI工具）。背景是蓝色的，中央有一个可爱的机器人图像。机器人周围有几种学习工具的图标，如烧瓶、笔、数学符号、笔记本等，这些图标代表科学、写作和数学等学科。图片主题是关于教育中使用人工智能工具。",
        "file_name": "ai.png"
    }
 ```
# TTS
## request: 
```json
        {
            "biz_info": {
                "sid": "987653",
                "cmd": "TTS",
                "params": "654"
            },
            "reqid":654,
            "content":"你好，中国我爱你222"

        }
```
## response:
```json
    {
        "userid": "15314684226",
        "reqid": 654,
        "event": {
            "sid": "987653",
            "event_type": "TTS_STREAM",
            "event_content": "你好，中国我爱你222\n",
            "event_params": "654|1|0"
        },
        "content": "",
        "file_name": "654_1_0"
    }
```
