from openai import OpenAI

client = OpenAI(
    api_key="452774f4094d0d544112582e407885a2:MTQ5MTc0YzBhM2JmOTdkNGFlN2UyZmZi",
    base_url="https://maas-coding-api.cn-huabei-1.xf-yun.com/v2"  # 可根据需要修改为其他兼容的API地址
)

stream = client.chat.completions.create(
    model="astron-code-latest",
    messages=[
        {"role": "user", "content": "今天天气怎么样?"}
    ],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content is not None:
        print(chunk.choices[0].delta.content, end="", flush=True)

print()  # 最后换行
