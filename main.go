package rapid

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "time"

    // Azzurro platform integrations - using mocks for now
    atp "./internal/atp"
    pod "./internal/pod"
    shepherd "./internal/shepherd"
    song "./internal/song"
)