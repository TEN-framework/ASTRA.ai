import { LanguageMap } from "@/common/constant";

export const voiceNameMap: LanguageMap = {
    "zh-CN": {
        azure: {
            male: "zh-CN-YunxiNeural",
            female: "zh-CN-XiaoxiaoNeural",
        },
        elevenlabs: {
            male: "pNInz6obpgDQGcFmaJgB", // Adam
            female: "Xb7hH8MSUJpSbSDYk0k2", // Alice
        },
        polly: {
            male: "Zhiyu",
            female: "Zhiyu",
        },
        openai: {
            male: "alloy",
            female: "shimmer"
        }
    },
    "en-US": {
        azure: {
            male: "en-US-BrianNeural",
            female: "en-US-JaneNeural",
        },
        elevenlabs: {
            male: "pNInz6obpgDQGcFmaJgB", // Adam
            female: "Xb7hH8MSUJpSbSDYk0k2", // Alice
        },
        polly: {
            male: "Matthew",
            female: "Ruth",
        },
        openai: {
            male: "alloy",
            female: "shimmer"
        }
    },
    "ja-JP": {
        azure: {
            male: "ja-JP-KeitaNeural",
            female: "ja-JP-NanamiNeural",
        },
        openai: {
            male: "alloy",
            female: "shimmer"
        }
    },
    "ko-KR": {
        azure: {
            male: "ko-KR-InJoonNeural",
            female: "ko-KR-JiMinNeural",
        },
        openai: {
            male: "alloy",
            female: "shimmer"
        }
    },
};

// Get the graph properties based on the graph name, language, and voice type
// This is the place where you can customize the properties for different graphs to override default property.json
export const getGraphProperties = (graphName: string, language: string, voiceType: string) => {
    let localizationOptions = {
        "greeting": "Hey, I\'m TEN Agent with OpenAI Realtime API Beta， anything I can help you with?",
        "checking_vision_text_items": "[\"Let me take a look...\",\"Let me check your camera...\",\"Please wait for a second...\"]",
    }

    if (language === "zh-CN") {
        localizationOptions = {
            "greeting": "TEN Agent 已连接，需要我为您提供什么帮助?",
            "checking_vision_text_items": "[\"让我看看你的摄像头...\",\"让我看一下...\",\"我看一下，请稍候...\"]",
        }
    } else if (language === "ja-JP") {
        localizationOptions = {
            "greeting": "TEN Agent に接続されました。今日は何をお手伝いしましょうか?",
            "checking_vision_text_items": "[\"ちょっと見てみます...\",\"カメラをチェックします...\",\"少々お待ちください...\"]",
        }
    } else if (language === "ko-KR") {
        localizationOptions = {
            "greeting": "TEN Agent 에이전트에 연결되었습니다. 오늘은 무엇을 도와드릴까요?",
            "checking_vision_text_items": "[\"조금만 기다려 주세요...\",\"카메라를 확인해 보겠습니다...\",\"잠시만 기다려 주세요...\"]",
        }
    }

    if (graphName == "camera.va.openai.azure") {
        return {
            "agora_rtc": {
                "agora_asr_language": language,
            },
            "openai_chatgpt": {
                "model": "gpt-4o",
                ...localizationOptions
            },
            "azure_tts": {
                "azure_synthesis_voice_name": voiceNameMap[language]["azure"][voiceType]
            }
        }
    } else if (graphName == "va.openai.v2v") {
        return {
            "openai_v2v_python": {
                "model": "gpt-4o-realtime-preview",
                "voice": voiceNameMap[language]["openai"][voiceType],
                "language": language,
                ...localizationOptions
            }
        }
    } else if (graphName == "va.openai.azure") {
        return {
            "agora_rtc": {
                "agora_asr_language": language,
            },
            "openai_chatgpt": {
                "model": "gpt-4o-mini",
                ...localizationOptions
            },
            "azure_tts": {
                "azure_synthesis_voice_name": voiceNameMap[language]["azure"][voiceType]
            }
        }
    } else if (graphName == "va.qwen.rag") {
        return {
            "agora_rtc": {
                "agora_asr_language": language,
            },
            "azure_tts": {
                "azure_synthesis_voice_name": voiceNameMap[language]["azure"][voiceType]
            }
        }
    }
    return {}
}
