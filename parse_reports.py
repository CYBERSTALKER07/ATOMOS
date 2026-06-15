import json
import glob
import os

transcript_dir = "/Users/shakhzod/.cursor/projects/Users-shakhzod-Desktop-V-O-I-D/agent-transcripts/064f053a-440e-4d5a-a45f-cca224aa78c2/subagents/*.jsonl"
files = glob.glob(transcript_dir)

for f in files:
    print(f"\n\n=================================\nFILE: {os.path.basename(f)}")
    with open(f, 'r') as fp:
        lines = fp.readlines()
        
    # Look for the last assistant message
    for line in reversed(lines):
        try:
            msg = json.loads(line)
            if msg.get("role") == "assistant":
                content = msg.get("message", {}).get("content", [])
                for block in content:
                    if block.get("type") == "text":
                        # If the text is just "[REDACTED]", maybe we look further back
                        text = block.get("text", "")
                        print(text)
                break # only the last assistant message
        except Exception as e:
            pass
