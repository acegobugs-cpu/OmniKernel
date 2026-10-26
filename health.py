from clientAi import client

def test_generation():
    print("🚀 Querying Cloud Generation Model...")
    
    completion = client.chat.completions.create(
        model="openai/gpt-oss-120b",
        messages=[
            {
                "role": "user",
                "content": "Give me a 1-sentence cheer for a developer who just successfully moved their AI stack to the cloud."
            }
        ]
    )
    
    answer = completion.choices[0].message.content
    print("✅ Success!")
    print(f"Cloud LLM Response: {answer.strip()}\n")

if __name__ == "__main__":
    print("--- Starting OmniKernel Cloud Infra Check ---\n")
    try:
        test_generation()
        print("--- Cloud systems operational! Your development speed is now optimized. ---")
    except Exception as e:
        print(f"❌ Error connecting to Cloud Provider: {e}")