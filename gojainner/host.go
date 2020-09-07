package gojainner

import (
    "sync"
    "github.com/dop251/goja"
    "github.com/dop251/goja_nodejs/require"
    "github.com/dop251/goja_nodejs/console"
    "fmt"
    "nbpy/utils"
)

type ScriptHost struct {
    program *goja.Program
    scriptDir string
    require *require.Registry
    nativeMods []ScriptModule

    mutexTime sync.Mutex
    asyncTime int64
    asyncTotal int64
    asyncTimeMax int64
    asyncTimeMin int64
}
type ScriptModule struct {
    Name string
    Loader require.ModuleLoader
}

func CreateScriptHost(sckDir string, mods... ScriptModule) (*ScriptHost, error) {
    host := new(ScriptHost)
    code := fmt.Sprintf(`%s/main.js`, sckDir)
    p, err := goja.Compile(code, nil, false)
    if err != nil {
        return nil, err
    }
    host.program = p
    host.scriptDir = sckDir
    host.require = new(require.Registry)
    host.nativeMods = mods

    return host, nil
}

func (s *ScriptHost) GetCallable(name string) (*goja.Runtime, goja.Callable, bool) {
    vm := s.CreateVM()

    _, err := vm.RunProgram(s.program)
    if err != nil {
        utils.LogError("Goja脚本执行出错", err)
        return nil, nil, false
    }
    vo := vm.Get("__main__")
    if goja.IsUndefined(vo) {
        utils.LogInfo("main is undefined")
        return nil, nil, false
    }

    o := vo.ToObject(vm)
    ofn := o.Get(name)
    if !goja.IsUndefined(vo) {
        fn, ok := goja.AssertFunction(ofn)
        if ok {
            return vm, fn, ok
        }
    }
    return nil, nil, false
}

func (s *ScriptHost) GetCallableWithProgram(p *goja.Program, name string) (*goja.Runtime, goja.Callable, bool) {
    vm := s.CreateVM()

    _, err := vm.RunProgram(p)
    if err != nil {
        utils.LogError("Goja脚本执行出错", err)
        return nil, nil, false
    }
    vo := vm.Get(name)
    if vo == nil || goja.IsUndefined(vo) {
        utils.LogInfo("%s is undefined", name)
        return nil, nil, false
    }

    fn, ok := goja.AssertFunction(vo)
    if ok {
        return vm, fn, ok
    }
    return nil, nil, false
}

func (s *ScriptHost) OnInitialize() {
    _ ,fn, ok := s.GetCallable("onInitialize")
    if ok {
        fn(goja.Undefined())
    }
}

func (s *ScriptHost) OnDestory() {
    _ ,fn, ok := s.GetCallable("onDestory")
    if ok {
        fn(goja.Undefined())
    }
}

func (s *ScriptHost) CreateVM() *goja.Runtime {
    vm := goja.New()
    s.require.Enable(vm)
    console.Enable(vm)
    for _, mod := range s.nativeMods {
        s.require.RegisterNativeModule(mod.Name, mod.Loader)
    }
    return vm
}
/*

func (s *ScriptHost) OnMessageIn(vm *goja.Runtime, msg map[string]interface{}) ([]interface{}, error) {
    _, err := vm.RunProgram(s.program)
    if err != nil {
        return nil, err
    }
    jsfn := vm.Get("onMessageIn")
    gofn, ok := goja.AssertFunction(jsfn)
    if ok {
        r, err := gofn(goja.Undefined(), vm.ToValue(msg))
        if err == nil {
            if goja.IsUndefined(r) {
                return nil, nil
            }
            rs := r.Export()
            grs, ok := rs.([]interface{})
            if ok {
                return grs, nil
            } else {
                return nil, nil
            }

        } else {
            return nil, err
        }
    }
    return nil, nil
}*/
