```
sequenceDiagram
    participant sdk as lookout-sdk
    participant DataService
    participant Bblfsh
    participant Analyzer
        sdk ->> +Analyzer: (gRPC) NotifyReviewEvent
        Analyzer ->> +DataService: (gRPC) GetChanges / GetFiles
        opt req.WantUAST?
            DataService ->> +Bblfsh: (gRPC) Parse
            Bblfsh -->> -DataService: uast.Node
        end
        DataService -->> -Analyzer: stream of Change / File
        Analyzer -->> -sdk: Comments
```

Use https://mermaidjs.github.io/mermaid-live-editor/ to render.

