package tengoinner

import (
    "script/tengo"
    "script/tengo/stdlib"
    "os"
    "fmt"
    "sync"
    "time"
)

type ScriptHost struct {
    compInst *tengo.Compiled
    mutexTime sync.Mutex
    asyncTime int64
    asyncTotal int64
    asyncTimeMax int64
    asyncTimeMin int64
}

type ScriptModule struct {
    Name string
    Module map[string] tengo.Object
}

func CreateScriptHost(sckDir string, mods ...ScriptModule) (*ScriptHost, error) {

    sckFile := fmt.Sprintf("%s/__init__.tengo", sckDir)
    fi, err := os.Stat(sckFile)
    if err != nil {
        return nil, err
    }
    f, err := os.Open(sckFile)
    if err != nil {
        return nil, err
    }
    code := make([]byte, fi.Size() * 2)
    n, err := f.Read(code)
    if err != nil {
        return nil, err
    }
    code = code[:n]
    modMap := tengo.NewModuleMap()
    modMap.AddMap(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

    for _, mod := range mods {
        modMap.AddBuiltinModule(mod.Name, mod.Module)
    }

    modMap.AddSourceModule("gs", code)

    scriptInit := tengo.NewScript([]byte(`gs := import("gs"); gs.Init()`))
    scriptInit.SetImports(modMap)
    scriptInit.SetImportDir(sckDir)
    scriptInit.EnableFileImport(true)
    _, err = scriptInit.Run()
    if err != nil {
        return nil, err
    }

    script := tengo.NewScript([]byte(`gs := import("gs"); gs.Run(message)`))
    script.SetImports(modMap)
    script.SetImportDir(sckDir)
    script.EnableFileImport(true)
    script.Add("message", nil)

    se := new(ScriptHost)
    se.compInst, err = script.Compile()
    return se, err
}

func RunScript(code string, sckDir string, mods ...ScriptModule) error {

    modMap := tengo.NewModuleMap()
    modMap.AddMap(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

    for _, mod := range mods {
        modMap.AddBuiltinModule(mod.Name, mod.Module)
    }

    scriptInit := tengo.NewScript([]byte(code))
    scriptInit.SetImports(modMap)
    scriptInit.SetImportDir(sckDir)
    scriptInit.EnableFileImport(true)
    _, err := scriptInit.Run()
    if err != nil {
        return err
    }

    return err
}

func CreateCompile(code string, sckDir string, mods ...ScriptModule) (*tengo.Compiled, error) {

    modMap := tengo.NewModuleMap()
    modMap.AddMap(stdlib.GetModuleMap(stdlib.AllModuleNames()...))

    for _, mod := range mods {
        modMap.AddBuiltinModule(mod.Name, mod.Module)
    }

    scriptInit := tengo.NewScript([]byte(code))
    scriptInit.SetImports(modMap)
    scriptInit.SetImportDir(sckDir)
    scriptInit.EnableFileImport(true)
    scriptInit.Add("argv", nil)
    scriptInit.Add("result", nil)
    comp, err := scriptInit.Compile()
    if err != nil {
        return nil, err
    }

    return comp, err
}


func (s *ScriptHost) OnMessageIn(comp *tengo.Compiled, msg map[string]interface{}) ([]interface{}, error) {
    st := time.Now().UnixNano()
    currCompInst := comp.Clone()
    err := currCompInst.Set("argv", msg)
    if err != nil {
        return nil, err
    }
    err = currCompInst.Run()
    result := currCompInst.Get("result")

    var ret []interface{} = nil
    if result.ValueType() == "array" {
        ret = result.Array()
    }
    currCompInst = nil
    s.recordAsyncTime(time.Now().UnixNano() - st)
    return ret, err
}

func (s *ScriptHost) recordAsyncTime(tv int64) {
    s.mutexTime.Lock()
    defer s.mutexTime.Unlock()
    s.asyncTime += tv
    s.asyncTotal += 1
    if tv > s.asyncTimeMax {
        s.asyncTimeMax = tv
    }
    if tv < s.asyncTimeMin {
        s.asyncTimeMin = tv
    } else if s.asyncTimeMin == 0 {
        s.asyncTimeMin = tv
    }
}
func (s *ScriptHost) GetAsyncInfo() (int64, int64, int64) {
    s.mutexTime.Lock()
    defer s.mutexTime.Unlock()
    if s.asyncTotal > 0 {
        return s.asyncTime / s.asyncTotal, s.asyncTimeMax, s.asyncTimeMin
    }
    return 0, s.asyncTimeMax, s.asyncTimeMin
}