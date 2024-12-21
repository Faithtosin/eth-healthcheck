package main

import (
    "bytes"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "time"
)

// EthSyncingRequest represents the JSON-RPC request payload for eth_syncing
type EthSyncingRequest struct {
    JSONRPC string      `json:"jsonrpc"`
    Method  string      `json:"method"`
    Params  []string    `json:"params"`
    ID      int         `json:"id"`
}

// EthSyncingResponse represents the JSON-RPC response for eth_syncing
// The result can be false if not syncing or an object if syncing.
type EthSyncingResponse struct {
    JSONRPC string           `json:"jsonrpc"`
    ID      int              `json:"id"`
    Result  interface{}      `json:"result"` // Can be bool (false) or an object
}

func main() {
    // Example: read the Ethereum node URL from environment variable (default: http://localhost:8545)
    ethNodeURL := os.Getenv("ETH_NODE_URL")
    if ethNodeURL == "" {
        ethNodeURL = "http://localhost:8545"
    }
    
    // Create an HTTP handler
    http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
        syncing, err := isNodeSyncing(ethNodeURL)
        if err != nil {
            log.Println("Error checking sync status:", err)
            // If we fail to check, we might treat it as unhealthy or return an error
            http.Error(w, "Error checking sync status", http.StatusInternalServerError)
            return
        }

        if syncing {
            // Node is syncing
            w.WriteHeader(http.StatusPartialContent) // 206
            w.Write([]byte("Node is still syncing"))
        } else {
            // Node is fully synced
            w.WriteHeader(http.StatusOK) // 200
            w.Write([]byte("Node is fully synced"))
        }
    })

    // Start the server (listening on port 8080 by default)
    log.Println("Starting healthcheck server on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

// isNodeSyncing calls the eth_syncing method on the given node
// and returns true if the node is syncing, or false otherwise.
func isNodeSyncing(ethNodeURL string) (bool, error) {
    // Prepare the JSON-RPC request
    requestBody, err := json.Marshal(EthSyncingRequest{
        JSONRPC: "2.0",
        Method:  "eth_syncing",
        Params:  []string{},
        ID:      1,
    })
    if err != nil {
        return false, err
    }

    // Create a POST request
    req, err := http.NewRequest(http.MethodPost, ethNodeURL, bytes.NewBuffer(requestBody))
    if err != nil {
        return false, err
    }
    req.Header.Set("Content-Type", "application/json")

    // Create an HTTP client with a timeout
    client := &http.Client{
        Timeout: 5 * time.Second,
    }

    // Execute the request
    resp, err := client.Do(req)
    if err != nil {
        return false, err
    }
    defer resp.Body.Close()

    // Parse the JSON-RPC response
    var ethSyncingResp EthSyncingResponse
    if err := json.NewDecoder(resp.Body).Decode(&ethSyncingResp); err != nil {
        return false, err
    }

    // The result can be `false` or an object
    switch v := ethSyncingResp.Result.(type) {
    case bool:
        // If it's a bool:
        //   false => node is not syncing
        //   true => theoretically means syncing, but eth_syncing typically returns an object if syncing
        return v, nil
    case map[string]interface{}:
        // If it's an object, the node is syncing
        return true, nil
    default:
        // Unexpected type
        return false, nil
    }
}
