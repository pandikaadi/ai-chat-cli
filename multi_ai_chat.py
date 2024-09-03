
import requests
import json
import os
# Set up the Claude API key and endpoint
os.environ.get("MULTI_AI_CLAUDE_API_KEY")
claude_api_key = os.environ.get("MULTI_AI_CLAUDE_API_KEY") 
claude_api_endpoint = os.environ.get("MULTI_AI_CLAUDE_EP") 
gemini_api_endpoint = os.environ.get("MULTI_AI_GEMINI_CREDS")
GREEN = '\033[92m'
BLUE = '\033[94m'
RESET = '\033[0m'
BLACK_TEXT = '\033[30m'
WHITE_BG = '\033[47m'
MODELS = []
def chat_with_llm():
    convo = []
    provider, model,data, headers = choose_model()

    while True:
        user_input = raw_input("You: ").format(GREEN)

        if user_input.lower() in ['exit', 'quit']:
            print("Exiting the chat.")
            break
        elif user_input.lower() == "change_model":
            provider, model, data, headers = choose_model()
            continue
        elif user_input.lower() == "clear_convo":
            convo = []
            continue
        convo.append(user_input)
        response = None
        reply = None
        formatted_convo = []
        if provider == 'claude': 
            for i, conv in enumerate(convo):
                role = 'assistant'
                if i % 2 == 0:
                    role = 'user'
                formatted_convo.append({"role": role, 'content': conv})
                data["messages"] = formatted_convo
            response = requests.post(claude_api_endpoint, json=data, headers=headers)

            if response.status_code == 200:
                result = json.loads(response.content.decode('utf-8'))
                reply = result['content'][0]['text']
        elif provider == 'gemini':
            for i, conv in enumerate(convo):
                role = 'model'
                if i % 2 == 0:
                    role = 'user'
                formatted_convo.append({"role": role, "parts": [{"text":conv}]})
                data["contents"] = formatted_convo
            response = requests.post(gemini_api_endpoint, json=data, headers=headers)

            if response.status_code == 200:
                result = json.loads(response.content.decode('utf-8'))
                reply = result['candidates'][0]['content']['parts'][0]['text']

        if response.status_code == 200: 
            print("{}{}{}{}".format(GREEN, provider + ": " + reply.encode('utf-8'), RESET, RESET ))
            convo.append(reply)
        else:
            print("Error:", response.status_code, response.text)


def choose_model():
    print(MODELS)
    model = None
    provider = None
    while model is None:
        model_input = raw_input("Choose model:")
        if model_input.lower() in ['exit', 'quit']:
            print("Exiting the chat.")
            break
        try :
            int_model_input = int(model_input)
            if int_model_input >= 0 and int_model_input < len(MODELS):
                model = MODELS[int_model_input]
                if "claude" in model:
                    provider = "claude"
                    data = {
                        "model": model,
                        "max_tokens": 300
                    } 
                    headers = {
                        "x-api-key": claude_api_key,
                        "anthropic-version": "2023-06-01",
                        "Content-Type": "application/json"
                    }
                else :
                    provider = "gemini"
                    headers = {
                        "Content-Type": "application/json"
                    }

                    data = {
                        "contents": None
                    }
            else :
                print("invalid model")
        except :
            print("invalid model")
     
    print("#### provider: " + provider + " | model:" + model)
    return provider, model, data, headers
 
if __name__ == "__main__":
    print(gemini_api_endpoint, claude_api_endpoint)
    if gemini_api_endpoint is not None:
        MODELS.append("gemini-1.5-flash")
    if claude_api_endpoint is not None or claude_api_key is not None:
        MODELS.append("claude-3-haiku-20240307")
        MODELS.append("claude-3-5-sonnet-20240620")
    if len(MODELS) == 0:
        print("no model available, please check your credentials")
    else: 
        chat_with_llm()
