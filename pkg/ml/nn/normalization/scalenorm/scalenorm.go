// Copyright 2019 spaGO Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scalenorm

import (
	"github.com/nlpodyssey/spago/pkg/mat"
	"github.com/nlpodyssey/spago/pkg/ml/ag"
	"github.com/nlpodyssey/spago/pkg/ml/nn"
	"log"
)

var (
	_ nn.Model     = &Model{}
	_ nn.Processor = &Processor{}
)

type Model struct {
	Gain *nn.Param `type:"weights"`
}

func New(size int) *Model {
	return &Model{
		Gain: nn.NewParam(mat.NewEmptyVecDense(size)),
	}
}

type Processor struct {
	opt   []interface{}
	model *Model
	mode  nn.ProcessingMode
	g     *ag.Graph
	gain  ag.Node
}

func (m *Model) NewProc(g *ag.Graph, opt ...interface{}) nn.Processor {
	p := &Processor{
		model: m,
		mode:  nn.Training,
		opt:   opt,
		g:     g,
		gain:  g.NewWrap(m.Gain),
	}
	p.init(opt)
	return p
}

func (p *Processor) init(opt []interface{}) {
	if len(opt) > 0 {
		log.Fatal("scalenorm: invalid init options")
	}
}

func (p *Processor) Model() nn.Model                { return p.model }
func (p *Processor) Graph() *ag.Graph               { return p.g }
func (p *Processor) RequiresFullSeq() bool          { return false }
func (p *Processor) Mode() nn.ProcessingMode        { return p.mode }
func (p *Processor) SetMode(mode nn.ProcessingMode) { p.mode = mode }

func (p *Processor) Forward(xs ...ag.Node) []ag.Node {
	ys := make([]ag.Node, len(xs))
	eps := p.g.NewScalar(1e-10)
	for i, x := range xs {
		norm := p.g.Sqrt(p.g.ReduceSum(p.g.Square(x)))
		ys[i] = p.g.Prod(p.g.DivScalar(x, p.g.AddScalar(norm, eps)), p.gain)
	}
	return ys
}
