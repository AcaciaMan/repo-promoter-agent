cURL example
The cURL example uses environment variables to store the agent’s endpoint ($AGENT_ENDPOINT) and access key ($AGENT_ACCESS_KEY). To return retrieval information about how the response was generated, such as the knowledge base data, guardrails, and functions used, the include_retrieval_info, include_guardrails_info, and the include_functions_info parameters to true in the request body are set to true.

curl -i \
  -X POST \
  $AGENT_ENDPOINT/api/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $AGENT_ACCESS_KEY" \
  -d '{
    "messages": [
      {
        "role": "user",
        "content": "What is the capital of France?"
      }
    ],
    "stream": false,
    "include_functions_info": true,
    "include_retrieval_info": true,
    "include_guardrails_info": true
  }'