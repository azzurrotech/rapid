package agent

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "time"
)

// ModelLoader handles loading and managing AI models for RAPID analysis
type ModelLoader struct {
    modelsDir string
    cacheDir  string
    ollamaURL string
    timeout   time.Duration
}

// ModelInfo represents metadata about an AI model
 type ModelInfo struct {
     Name        string   `json:"name"`
     Description string   `json:"description"`
     SizeMB      int      `json:"size_mb"`
     Parameters  string   `json:"parameters"`
     ContextSize int      `json:"context_size"`
     Quant       string   `json:"quant"`
     URL         string   `json:"url"`
 }

// ModelInfo represents metadata about an AI model
 type ModelInfo struct {
     Name        string   `json:"name"`
     Description string   `json:"description"`
     SizeMB      int      `json:"size_mb"`
     Parameters  string   `json:"parameters"`
     ContextSize int      `json:"context_size"`
     Quant       string   `json:"quant"`
     URL         string   `json:"url"`
 }

// ModelLoadResult represents the result of loading a model
 type ModelLoadResult struct {
     Model      ModelInfo
     Success    bool
     Error      string
     LoadTime   time.Duration
     TokenCount int
 }

// NewModelLoader creates a new model loader
 func NewModelLoader(modelsDir, cacheDir, ollamaURL string, timeout time.Duration) *ModelLoader {
     if modelsDir == "" {
         modelsDir = "./models"
     }
     if cacheDir == "" {
         cacheDir = "./cache"
     }
     if ollamaURL == "" {
         ollamaURL = "http://localhost:11434"
     }
     if timeout == 0 {
         timeout = 30 * time.Second
     }

     // Create directories
     os.MkdirAll(modelsDir, 0755)
     os.MkdirAll(cacheDir, 0755)

     return &ModelLoader{
         modelsDir: modelsDir,
         cacheDir:  cacheDir,
         ollamaURL: ollamaURL,
         timeout:   timeout,
     }
 }

// GetAvailableModels lists all available models
func (ml *ModelLoader) GetAvailableModels() ([]ModelInfo, error) {
    models := []ModelInfo{}

    if _, err := os.Stat(ml.modelsDir); os.IsNotExist(err) {
        return models, nil
    }

    files, err := ioutil.ReadDir(ml.modelsDir)
    if err != nil {
        return nil, fmt.Errorf("failed to read models directory: %w", err)
    }

    for _, file := range files {
        if file.IsDir() {
            modelPath := filepath.Join(ml.modelsDir, file.Name())
            info, err := ml.loadModelInfo(modelPath)
            if err != nil {
                log.Printf("Failed to load model info for %s: %v", file.Name(), err)
                continue
            }
            models = append(models, info)
        }
    }

    return models, nil
}

// loadModelInfo loads model metadata from directory
func (ml *ModelLoader) loadModelInfo(modelPath string) (ModelInfo, error) {
    info := ModelInfo{}

    // Check for config file
    configFile := filepath.Join(modelPath, "config.json")
    if _, err := os.Stat(configFile); err == nil {
        data, err := ioutil.ReadFile(configFile)
        if err != nil {
            return info, fmt.Errorf("failed to read config file: %w", err)
        }

        err = json.Unmarshal(data, &info)
        if err != nil {
            return info, fmt.Errorf("failed to parse config file: %w", err)
        }
        return info, nil
    }

    // Try to infer from directory structure
    info.Name = filepath.Base(modelPath)
    info.Description = fmt.Sprintf("Ollama model: %s", info.Name)
    info.SizeMB = 0
    info.Parameters = "unknown"
    info.ContextSize = 0
    info.Quant = ""
    info.URL = ml.ollamaURL + "/api/tags"

    return info, nil
}

// LoadModel loads a specific model
func (ml *ModelLoader) LoadModel(modelName string) (ModelLoadResult, error) {
    start := time.Now()
    result := ModelLoadResult{Success: false}

    modelPath := filepath.Join(ml.modelsDir, modelName)
    if _, err := os.Stat(modelPath); os.IsNotExist(err) {
        result.Error = fmt.Sprintf("model directory not found: %s", modelName)
        result.Model = ModelInfo{Name: modelName, Description: result.Error}
        return result, nil
    }

    // Try to load from Ollama API first
    if ml.tryOllamaLoad(modelName) {
        result.Success = true
        result.Model = ModelInfo{Name: modelName, Description: "Loaded from Ollama"}
    } else {
        // Fallback to local loading
        result = ml.loadLocalModel(modelName)
    }

    result.LoadTime = time.Since(start)
    return result, nil
}

// tryOllamaLoad attempts to load model from Ollama API
func (ml *ModelLoader) tryOllamaLoad(modelName string) bool {
    // This is a simplified model - in reality you'd make HTTP requests
    // to the Ollama API to check if model is available
    modelPath := filepath.Join(ml.modelsDir, modelName)

    // Check if model directory exists and contains files
    if files, err := os.ReadDir(modelPath); err == nil && len(files) > 0 {
        return true
    }

    return false
}

// loadLocalModel loads a model from local files
func (ml *ModelLoader) loadLocalModel(modelName string) ModelLoadResult {
    modelPath := filepath.Join(ml.modelsDir, modelName)

    // Count total tokens by reading model files
    tokenCount := 0
    fileCount := 0

    filepath.Walk(modelPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return nil
        }
        if !info.IsDir() {
            fileCount++
            // Simple token counting (rough estimate)
            if data, err := ioutil.ReadFile(path); err == nil {
                tokenCount += len(data) / 4 // Rough estimate
            }
        }
        return nil
    })

    return ModelLoadResult{
        Model: ModelInfo{Name: modelName, Description: "Locally cached model"},
        Success:    true,
        Error:      "",
        LoadTime:   0,
        TokenCount: tokenCount,
    }
}

// getOllamaTags fetches model tags from Ollama
func (ml *ModelLoader) getOllamaTags() ([]ModelInfo, error) {
    // In a real implementation, this would make HTTP requests
    // For now, return empty slice as this is a starter implementation
    return []ModelInfo{}, nil
}

// ModelCache represents cached model information
 type ModelCache struct {
     Models   []ModelInfo
     LastUpdate time.Time
     TTL       time.Duration
 }

// NewModelCache creates a new model cache
 func NewModelCache(ttl time.Duration) *ModelCache {
     if ttl == 0 {
         ttl = 24 * time.Hour
     }

     return &ModelCache{
         Models:   []ModelInfo{},
         LastUpdate: time.Now(),
         TTL:       ttl,
     }
 }

// AddModel adds a model to the cache
 func (mc *ModelCache) AddModel(model ModelInfo) {
     mc.Models = append(mc.Models, model)
     mc.LastUpdate = time.Now()
 }

// GetModels returns all cached models
 func (mc *ModelCache) GetModels() []ModelInfo {
     return mc.Models
 }

// IsExpired checks if the cache is expired
 func (mc *ModelCache) IsExpired() bool {
     return time.Since(mc.LastUpdate) > mc.TTL
 }