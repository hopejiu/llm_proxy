from openai import OpenAI

client = OpenAI(
    api_key="4",
    base_url="http://localhost:8888/v1/"  # 可根据需要修改为其他兼容的API地址
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
