package graphviz

import (
	"fmt"
	"gitee.com/MetaphysicCoding/light-flow/core"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"sync"
)

const (
	GraphPrefix = "graph_"
)

var (
	lock   = sync.RWMutex{}
	graphs = map[string]any{}
)

func Decorate(f func(info *core.ProcedureInfo)) func(info *core.ProcedureInfo) {
	return func(info *core.ProcedureInfo) {
		DrawByProcedureInfo(info)
		f(info)
	}
}

func DrawByProcedureInfo(info *core.ProcedureInfo) {
	lock.RLock()
	_, exist := graphs[info.Name]
	lock.RUnlock()
	if exist {
		return
	}
	lock.Lock()
	defer lock.Unlock()
	_, exist = graphs[info.Name]
	// DCL
	if exist {
		return
	}
	gv := graphviz.New()
	graph, err := gv.Graph()
	if err != nil {
		return
	}
	nodes := make(map[string]*cgraph.Node, len(info.StepMap))
	for name := range info.StepMap {
		node, _ := graph.CreateNode(name)
		nodes[name] = node
	}

	for _, step := range info.StepMap {
		for _, next := range step.Next {
			addEdge(graph, nodes[step.Name], nodes[next])
		}
	}

	graph.SetLayout("dot")
	// 保存为图片
	err = gv.RenderFilename(graph, graphviz.PNG, fmt.Sprintf("%s%s.png", GraphPrefix, info.Name))
}

func addEdge(graph *cgraph.Graph, from, to *cgraph.Node) {
	edge, _ := graph.CreateEdge("", from, to)
	edge.SetDir(cgraph.ForwardDir) // 设置有向边
}
