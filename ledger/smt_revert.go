/**
 *  @file
 *  @copyright defined in aergo/LICENSE.txt
 */

package ledger

import (
	"bytes"
)

// maybeDeleteSubTree compares the subtree nodes of 2 tries and keeps only the older one
func (s *SMT) maybeDeleteSubTree(original, maybeDelete []byte, height, iBatch int, batch, batch2 [][]byte, ch chan<- (error)) {
	if bytes.Equal(original, maybeDelete) || len(maybeDelete) == 0 {
		ch <- nil
		return
	}
	if height == 0 {
		ch <- nil
		return
	}

	// if this point os reached, then the root of the batch is same
	// so the batch is also same.
	batch, iBatch, lnode, rnode, isShortcut, lerr := s.loadChildren(original, height, iBatch, batch)
	if lerr != nil {
		ch <- lerr
		return
	}
	batch2, _, lnode2, rnode2, isShortcut2, rerr := s.loadChildren(maybeDelete, height, iBatch, batch2)
	if rerr != nil {
		ch <- rerr
		return
	}

	if isShortcut != isShortcut2 {
		if isShortcut {
			ch1 := make(chan error, 1)
			s.deleteSubTree(maybeDelete, height, iBatch, batch2, ch1)
			err := <-ch1
			if err != nil {
				ch <- err
				return
			}
		} else {
			s.maybeDeleteRevertedNode(maybeDelete, iBatch)
		}
	} else {
		if isShortcut {
			if !bytes.Equal(lnode, lnode2) || !bytes.Equal(rnode, rnode2) {
				s.maybeDeleteRevertedNode(maybeDelete, iBatch)
			}
		} else {
			// Delete subtree if not equal
			s.maybeDeleteRevertedNode(maybeDelete, iBatch)
			ch1 := make(chan error, 1)
			ch2 := make(chan error, 1)
			go s.maybeDeleteSubTree(lnode, lnode2, height-1, 2*iBatch+1, batch, batch2, ch1)
			go s.maybeDeleteSubTree(rnode, rnode2, height-1, 2*iBatch+2, batch, batch2, ch2)
			err1 := <-ch1
			err2 := <-ch2
			if err1 != nil {
				ch <- err1
				return
			}
			if err2 != nil {
				ch <- err2
				return
			}
		}
	}
	ch <- nil
}

// deleteSubTree deletes all the nodes contained in a tree
func (s *SMT) deleteSubTree(root []byte, height, iBatch int, batch [][]byte, ch chan<- (error)) {
	if height == 0 || len(root) == 0 {
		ch <- nil
		return
	}
	batch, iBatch, lnode, rnode, isShortcut, err := s.loadChildren(root, height, iBatch, batch)
	if err != nil {
		ch <- err
		return
	}
	if !isShortcut {
		ch1 := make(chan error, 1)
		ch2 := make(chan error, 1)
		go s.deleteSubTree(lnode, height-1, 2*iBatch+1, batch, ch1)
		go s.deleteSubTree(rnode, height-1, 2*iBatch+2, batch, ch2)
		lerr := <-ch1
		rerr := <-ch2
		if lerr != nil {
			ch <- lerr
			return
		}
		if rerr != nil {
			ch <- rerr
			return
		}
	}
	s.maybeDeleteRevertedNode(root, iBatch)
	ch <- nil
}

// maybeDeleteRevertedNode adds the node to updatedNodes to be reverted
func (s *SMT) maybeDeleteRevertedNode(root []byte, iBatch int) {
	if iBatch == 0 {
		s.db.revertMux.Lock()
		s.db.nodesToRevert = append(s.db.nodesToRevert, root)
		s.db.revertMux.Unlock()
	}
}
