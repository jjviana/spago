// Copyright 2019 spaGO Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package nn

import (
	"saientist.dev/spago/pkg/mat"
	"saientist.dev/spago/pkg/ml/ag"
	"sync"
)

// Linear performs a linear transformation of the type Wx.
func Linear(g *ag.Graph, w, x ag.Node) ag.Node {
	return g.Mul(w, x)
}

// Affine performs an affine transformation over an arbitrary (odd) number of nodes held in the input.
// The first node is the “bias”, which is added to the output as-is.
// The remaining nodes of the form "Wx" are multiplied together in pairs, then added.
// The pairs except the first whose "x" is nil are not considered.
// y = b + W1x1 + W2x2 + ... + WnXn
func Affine(g *ag.Graph, xs ...ag.Node) ag.Node {
	if len(xs)%2 == 0 {
		panic("nn: the number of arguments of the affine transformation should be odd")
	}
	y := g.Add(xs[0], Linear(g, xs[1], xs[2])) // b + Wx
	for i := 3; i < len(xs)-1; i += 2 {
		w := xs[i]
		x := xs[i+1]
		if x != nil {
			y = g.Add(y, Linear(g, w, x))
		}
	}
	return y
}

// BiLinear performs a bilinear transformation of the type (x_1 W x_2)
func BiLinear(g *ag.Graph, w, x1, x2 ag.Node) ag.Node {
	return g.Mul(g.Mul(g.T(x1), w), x2)
}

// BiAffine performs a biaffine transformation.
func BiAffine(g *ag.Graph, w, u, v, b, x1, x2 ag.Node) ag.Node {
	return g.Add(g.Add(g.Add(BiLinear(g, w, x1, x2), g.Mul(g.T(u), x1)), g.Mul(g.T(v), x2)), b)
}

// Conv2D performs a 2D convolution.
func Conv2D(g *ag.Graph, w, x ag.Node, xStride, yStride int) ag.Node {
	var dimx, dimy int
	if (x.Value().Rows()-w.Value().Rows())%xStride != 0 {
		panic("Incompatible stride value for rows")
	}
	if (x.Value().Columns()-w.Value().Columns())%yStride != 0 {
		panic("Incompatible stride value for columns")
	}
	dimx = (x.Value().Rows()-w.Value().Rows())/xStride + 1
	dimy = (x.Value().Columns()-w.Value().Columns())/yStride + 1

	var outList []ag.Node
	for i := 0; i < dimx; i++ {
		for j := 0; j < dimy; j++ {
			var view = g.View(x, i*xStride, j*yStride, w.Value().Rows(), w.Value().Columns())
			var dotProduct = g.Dot(view, w)
			outList = append(outList, dotProduct)
		}
	}

	return g.Reshape(g.Concat(outList...), dimx, dimy)
}

// ScaledDotProductAttention is a self-attention mechanism relating different positions of a single sequence in order to compute a representation of the same sequence.
// This method requires that the query, the key and the value vectors have already been obtained from the input sequence.
// The scaled factor is the square root of the dimension of the key vectors.
func ScaledDotProductAttention(g *ag.Graph, qs, ks, vs []ag.Node, scaledFactor float64) (context []ag.Node, probs []mat.Matrix) {
	context = make([]ag.Node, len(qs))
	probs = make([]mat.Matrix, len(qs))
	keys := g.Stack(ks...)
	values := g.T(g.Stack(vs...))
	divTerm := g.NewScalar(scaledFactor)
	for i, q := range qs {
		attScores := g.DivScalar(g.Mul(keys, q), divTerm)
		attProbs := g.Softmax(attScores)
		context[i] = g.Mul(values, attProbs)
		probs[i] = attProbs.Value()
	}
	return
}

// ScaledDotProductAttentionConcurrent does the same thing as ScaledDotProductAttention but processes input concurrently.
func ScaledDotProductAttentionConcurrent(g *ag.Graph, qs, ks, vs []ag.Node, scaledFactor float64) (context []ag.Node, probs []mat.Matrix) {
	context = make([]ag.Node, len(qs))
	probs = make([]mat.Matrix, len(qs))
	keys := g.Stack(ks...)
	values := g.T(g.Stack(vs...))
	divTerm := g.NewScalar(scaledFactor)
	var wg sync.WaitGroup
	wg.Add(len(qs))
	for i, q := range qs {
		go func(i int, q ag.Node) {
			defer wg.Done()
			attScores := g.DivScalar(g.Mul(keys, q), divTerm)
			attProbs := g.Softmax(attScores)
			context[i] = g.Mul(values, attProbs)
			probs[i] = attProbs.Value()
		}(i, q)
	}
	wg.Wait()
	return
}