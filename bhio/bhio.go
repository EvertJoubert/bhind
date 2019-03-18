package bhio

import (
	"fmt"
	"io"
	"os"
	"sync"
)

type IReader interface {
	ReadAllToHandle(hndle func([]byte) (int, error)) (err error)
	ReadAll(io.Writer) error
}

//BhIORW Reader Writer
type BhIORW struct {
	//altReader      io.Reader
	//altCloser      io.Closer
	//altReaderLock  *sync.Mutex
	//altReadSeeker  io.ReadSeeker
	//altWriter      io.Writer
	//altWriterLock  *sync.Mutex
	//altWriteSeeker io.WriteSeeker
	io.Reader
	io.Writer
	io.ReadSeeker
	io.WriteSeeker
	buffer [][]byte
	bytes  []byte
	bytesi int
	fpath  string
	finfo  os.FileInfo
	cached bool
	altW   io.Writer
	cur    *BhReadWriteCursor
}

func (ioRW *BhIORW) UnderlyingCursor() *BhReadWriteCursor {
	return ioRW.cur
}

func (ioRW *BhIORW) SeekIndex() int64 {
	if ioRW.cur == nil {
		return -1
	} else {
		return ioRW.cur.SeekIndex()
	}
}

func (ioRW *BhIORW) FileInfo() os.FileInfo {
	return ioRW.finfo
}

func NewBhIORW(a ...interface{}) (bhioRW *BhIORW, err error) {
	bhioRW = &BhIORW{cached: false}
	if len(a) > 0 {
		for _, d := range a {
			if fpath, fpathok := d.(string); fpathok {
				bhioRW.fpath = fpath
			} else if finfofound, finfofoundOk := d.(os.FileInfo); finfofoundOk {
				bhioRW.finfo = finfofound
				if finfofound.Size() < int64(2*1024*1024) {
					bhioRW.cached = true
				}
			} else if cached, cachedok := d.(bool); cachedok {
				bhioRW.cached = cached
			} else if altW, altWok := d.(io.Writer); altWok {
				bhioRW.altW = altW
			}
		}
	}

	if bhioRW.cached && bhioRW.fpath != "" && bhioRW.finfo != nil {
		if f, ferr := os.Open(bhioRW.fpath); ferr == nil {
			bhioRW.Print(f)
		} else {
			err = ferr
		}
	}
	return bhioRW, err
}

//Size actual size of BhIORW Content
func (ioRW *BhIORW) Size() (size int64) {
	if ioRW.finfo != nil {
		size = ioRW.finfo.Size()
	} else {
		if ioRW.Empty() {
			size = 0
		} else {
			size = ioRW.BufferSize() + ioRW.NonBufferSize()
		}
	}
	return size
}

//NonBufferSize Size of BhIORW Content excluding the buffer
func (ioRW *BhIORW) NonBufferSize() (nonsize int64) {
	if ioRW.bytesi > 0 {
		nonsize = int64(ioRW.bytesi)
	} else {
		nonsize = 0
	}
	return nonsize
}

//BufferSize only buffer size of BhIORW Content
func (ioRW *BhIORW) BufferSize() (bufsize int64) {
	if len(ioRW.buffer) == 0 {
		bufsize = 0
	} else {
		bufsize = int64(len(ioRW.buffer)) * int64(len(ioRW.buffer[0]))
	}
	return bufsize
}

//ReadAllToHandle perform same action as ReaadAll just calling a custom handle as an ouput (write) caller
func (ioRW *BhIORW) ReadAllToHandle(hndle func([]byte) (int, error)) (err error) {
	if hndle == nil {
		err = fmt.Errorf("No callable handle assigend")
	} else {
		buf := make([]byte, 4096)
		for {
			if nb, nberr := ioRW.Read(buf); nberr == nil || nberr == io.EOF {
				if nb > 0 {
					bi := 0
					for bi < nb {
						if wn, werr := hndle(buf[bi : bi+(nb-bi)]); werr == nil {
							if wn > 0 {
								bi += wn
							}
						} else {
							nberr = werr
							break
						}
					}
				}
				if nberr == io.EOF || nberr != nil {
					err = nberr
					break
				}
			} else {
				err = nberr
				break
			}
		}
	}
	return err
}

//ReadAll content from IORW into w io.Writer
func (ioRW *BhIORW) ReadAll(w io.Writer) (err error) {
	if len(ioRW.buffer) > 0 {
		for _, iob := range ioRW.buffer {
			if _, errw := writeBytesToWriter(iob, w); errw != nil {
				err = errw
			}
			if err != nil {
				break
			}
		}
		if err == nil && ioRW.bytesi > 0 {
			_, err = writeBytesToWriter(ioRW.bytes[0:ioRW.bytesi], w)
		}
	}
	return err
}

func readBytesFromReader(b []byte, r io.Reader) (n int, err error) {
	if bl := len(b); bl > 0 {
		bi := 0
		for bi < bl {
			nr, errr := r.Read(b[bi : bi+(bl-bi)])
			if errr != nil {
				err = errr
			}
			if nr > 0 {
				bi += nr
				n += nr
			}
			if err != nil {
				break
			}
			if nr == 0 {
				err = io.EOF
				break
			}
		}
	}
	return n, err
}

func writeBytesToWriter(b []byte, w io.Writer) (n int, err error) {
	if bl := len(b); bl > 0 {
		bi := 0
		for bi < bl {
			nw, errw := w.Write(b[bi : bi+(bl-bi)])
			if errw != nil {
				err = errw
			}
			if nw > 0 {
				bi += nw
				n += nw
			}
			if err != nil {
				break
			}
		}
	}
	return n, err
}

//Read into b []byte n amount of bytes from BhIORW
func (ioRW *BhIORW) Read(p []byte) (n int, err error) {
	if ioRW.cur == nil {
		if ioRW.Size() == 0 {
			n = 0
			err = io.EOF
		} else {
			ioRW.cur = ioRW.ReadWriteCursor(true)
			n, err = ioRW.cur.Read(p)
		}
	} else {
		n, err = ioRW.cur.Read(p)
	}
	return n, err
}

//WriteAll content from r io.Reader
func (ioRW *BhIORW) WriteAll(r io.Reader) (err error) {
	bts := make([]byte, 4096)
	for {
		if rn, rerr := readBytesFromReader(bts, r); rerr == nil || rerr == io.EOF {
			if rn > 0 {
				if _, werr := writeBytesToWriter(bts[0:rn], ioRW); werr != nil {
					err = werr
					break
				}
			} else {
				break
			}
			if rerr == io.EOF {
				break
			}
		} else {
			err = rerr
			break
		}
	}
	return err
}

//Seek -> refer to io.ReadSeeker go documentation
func (ioRW *BhIORW) Seek(offset int64, whence int) (n int64, err error) {
	if ioRW.cur != nil {
		n, err = ioRW.cur.Seek(offset, whence)
	}
	return n, err
}

//Write b []byte n amount of bytes into BhIORW
func (ioRW *BhIORW) Write(p []byte) (n int, err error) {
	if pl := len(p); pl > 0 {
		for n < pl {
			if ioRW.altW == nil {
				if len(ioRW.bytes) == 0 {
					ioRW.bytes = make([]byte, 4096)
					ioRW.bytesi = 0
				}
				if (pl - n) >= (4096 - ioRW.bytesi) {
					copy(ioRW.bytes[ioRW.bytesi:ioRW.bytesi+(4096-ioRW.bytesi)], p[n:n+(4096-ioRW.bytesi)])
					n += (4096 - ioRW.bytesi)
					ioRW.bytesi += (4096 - ioRW.bytesi)
				} else {
					copy(ioRW.bytes[ioRW.bytesi:ioRW.bytesi+(pl-n)], p[n:n+(pl-n)])
					ioRW.bytesi += (pl - n)
					n += (pl - n)
				}
				if len(ioRW.bytes) == ioRW.bytesi {
					if ioRW.buffer == nil {
						ioRW.buffer = [][]byte{}
					}
					ioRW.buffer = append(ioRW.buffer, ioRW.bytes[:])
					ioRW.bytes = nil
					ioRW.bytesi = 0
				}
			} else {
				if len(ioRW.bytes) == 0 {
					ioRW.bytes = make([]byte, 4096)
					ioRW.bytesi = 0
				}
				if (pl - n) >= (4096 - ioRW.bytesi) {
					copy(ioRW.bytes[ioRW.bytesi:ioRW.bytesi+(4096-ioRW.bytesi)], p[n:n+(4096-ioRW.bytesi)])
					n += (4096 - ioRW.bytesi)
					ioRW.bytesi += (4096 - ioRW.bytesi)
				} else {
					copy(ioRW.bytes[ioRW.bytesi:ioRW.bytesi+(pl-n)], p[n:n+(pl-n)])
					ioRW.bytesi += (pl - n)
					n += (pl - n)
				}
				if ioRW.bytesi > 0 {
					wi := 0
					for wi < ioRW.bytesi {
						nw, nwerr := ioRW.altW.Write(ioRW.bytes[wi : ioRW.bytesi-wi])
						wi += nw
						if nwerr != nil {
							err = nwerr
							break
						}
					}
					ioRW.bytesi = 0
				}
			}
		}
	}
	return n, err
}

//String allow for any BhIORW instance to be printed e.g fmt.Fprint
func (ioRW *BhIORW) String() (s string) {
	s = ""
	if len(ioRW.buffer) > 0 {
		for b := range ioRW.buffer {
			s += string(b)
		}
	}
	if ioRW.bytesi > 0 {
		s += string(ioRW.bytes[0:ioRW.bytesi])
	}
	return s
}

//Close or cleanup IORW
func (ioRW *BhIORW) Close() (err error) {
	if ioRW.buffer != nil {
		for len(ioRW.buffer) > 0 {
			ioRW.buffer[0] = nil
			if len(ioRW.buffer) > 1 {
				ioRW.buffer = ioRW.buffer[1:]
			} else {
				break
			}
		}
		ioRW.buffer = nil
	}
	if ioRW.bytes != nil {
		ioRW.bytes = nil
	}
	if ioRW.bytesi > 0 {
		ioRW.bytesi = 0
	}
	if ioRW.altW != nil {
		ioRW.altW = nil
	}
	if ioRW.cur != nil {
		ioRW.cur.Close()
		ioRW.cur = nil
	}
	return err
}

//Print -. conveniant method that works the same as fmt.Fprint but writing to BhIORW
func (ioRW *BhIORW) Print(a ...interface{}) (n int, err error) {
	for _, d := range a {
		if refiorw, refiorwok := d.(*BhIORW); refiorwok {
			if len(refiorw.buffer) > 0 {
				for _, b := range refiorw.buffer {
					ioRW.Write(b)
				}
			}
			if refiorw.bytesi > 0 {
				ioRW.Write(refiorw.bytes[0:refiorw.bytesi])
			}
		} else if ior, iorok := d.(io.Reader); iorok {
			buf := make([]byte, 4096)
			for {
				nr, nrerr := ior.Read(buf)
				if nrerr == nil || nrerr == io.EOF {
					if nr > 0 {
						ioRW.Write(buf[0:nr])
					}
				}
				if nrerr != nil || nr == 0 {
					break
				}
			}
		} else {
			if d != nil {
				if uintb, uintbok := d.([]uint8); uintbok {
					fmt.Fprint(ioRW, string(uintb))
				} else if bb, bbok := d.([]byte); bbok {
					fmt.Fprint(ioRW, string(bb))
				} else {
					fmt.Fprint(ioRW, d)
				}
			}
		}
	}
	return n, err
}

//Println -. conveniant method that works the same as fmt.Fprintln but writing to BhIORW
func (ioRW *BhIORW) Println(a ...interface{}) (n int, err error) {
	if n, err = ioRW.Print(a...); err == nil {
		n, err = fmt.Fprintln(ioRW)
	}
	return n, err
}

//ReadWriteCursor create a cursor instance to handle Read operations in multi session environments
func (ioRW *BhIORW) ReadWriteCursor(enableLocking bool) (ioRWCur *BhReadWriteCursor) {
	suggestedMaxChunkSize := ioRW.Size()
	suggestedMaxBufSize := int64(4096)

	if suggestedMaxChunkSize > int64(1*1024*1024) {
		suggestedMaxBufSize = int64(1 * 1024 * 1024)
	}

	if suggestedMaxBufSize > suggestedMaxChunkSize {
		suggestedMaxBufSize = suggestedMaxChunkSize
	}
	ioRWCur = &BhReadWriteCursor{ioRW: ioRW, bytesi: 0, seekoffindex: 0, maxBufferSize: suggestedMaxBufSize}
	if !ioRW.cached && ioRW.finfo != nil && ioRW.fpath != "" {
		if f, ferr := os.Open(ioRW.fpath); ferr == nil {
			ioRWCur.bufReadCloser = f
			ioRWCur.bufReadSeeker = f
		}
	}
	if enableLocking {
		ioRWCur.lock = &sync.Mutex{}
	}
	return ioRWCur
}

func (ioRW *BhIORW) cursorNextReadBytes(ioRWCur *BhReadWriteCursor) (nextbytes []byte) {
	if iorws := ioRW.Size(); iorws > 0 && ioRWCur.seekoffindex < iorws {
		if ioRW.finfo != nil && !ioRW.cached {
			if ioRWCur.maxBuffer == nil {
				if ioRWCur.maxBufferLastSeekIndex >= iorws {
					return nextbytes
				}
				ioRWCur.maxBuffer, _ = NewBhIORW()
				ioRWCur.maxBufferSeekIndex = 0
				currentMaxBufSize := int64(0)
				currentMaxBytes := make([]byte, ioRWCur.maxBufferSize)

				doneMaxRead := make(chan bool, 1)
				go func() {
					for currentMaxBufSize < ioRWCur.maxBufferSize {
						for ioRWCur.maxBufferLastSeekIndex < iorws {
							ncmr, ncmrerr := ioRWCur.bufReadSeeker.Read(currentMaxBytes[0 : ioRWCur.maxBufferSize-currentMaxBufSize])
							if ncmr > 0 {
								ncmw, ncmwerr := ioRWCur.maxBuffer.Write(currentMaxBytes[0:ncmr])
								if ncmw > 0 {
									currentMaxBufSize += int64(ncmw)
									ioRWCur.maxBufferLastSeekIndex += int64(ncmw)
								}
								if ncmwerr != nil {
									if ncmrerr == nil {
										ncmrerr = ncmwerr
									}
								}
							}
							if ncmrerr != nil {
								break
							}
							if currentMaxBufSize >= ioRWCur.maxBufferSize {
								break
							}
						}
					}
					doneMaxRead <- true
				}()
				<-doneMaxRead
			}
			if ioRWCur.maxBuffer != nil {
				if iorwcurbufs := ioRWCur.maxBuffer.BufferSize(); ioRWCur.maxBufferSeekIndex < iorwcurbufs {
					bufl := int64(len(ioRWCur.maxBuffer.buffer[0]))
					bufi := int((ioRWCur.maxBufferSeekIndex - (ioRWCur.maxBufferSeekIndex % bufl)) / bufl)
					bytes := ioRWCur.maxBuffer.buffer[bufi]
					nextbytes = bytes[int(ioRWCur.maxBufferSeekIndex%bufl):]
					ioRWCur.maxBufferSeekIndex += int64(len(nextbytes))
				} else if iorwcurs := ioRWCur.maxBuffer.Size(); ioRWCur.maxBufferSeekIndex < iorwcurs {
					if iorwcurbufs > 0 {
						nextbytes = ioRWCur.maxBuffer.bytes[int(ioRWCur.maxBufferSeekIndex-iorwcurbufs):ioRWCur.maxBuffer.bytesi]
					} else {
						nextbytes = ioRWCur.maxBuffer.bytes[int(ioRWCur.maxBufferSeekIndex):ioRWCur.maxBuffer.bytesi]
					}
					ioRWCur.maxBufferSeekIndex += int64(len(nextbytes))
				} else {
					nextbytes = nil
				}
				if ioRWCur.maxBufferSeekIndex >= ioRWCur.maxBufferSize {
					ioRWCur.maxBuffer.Close()
					ioRWCur.maxBuffer = nil
					ioRWCur.maxBufferSeekIndex = 0
				}
			}
		} else {
			if iorwbufs := ioRW.BufferSize(); ioRWCur.seekoffindex < iorwbufs {
				bufl := int64(len(ioRW.buffer[0]))
				bufi := int((ioRWCur.seekoffindex - (ioRWCur.seekoffindex % bufl)) / bufl)
				bytes := ioRW.buffer[bufi]
				nextbytes = bytes[int(ioRWCur.seekoffindex%bufl):]
			} else if ioRWCur.seekoffindex < iorws {
				if iorwbufs > 0 {
					nextbytes = ioRW.bytes[int(ioRWCur.seekoffindex-iorwbufs):ioRW.bytesi]
				} else {
					nextbytes = ioRW.bytes[int(ioRWCur.seekoffindex):ioRW.bytesi]
				}
			} else {
				nextbytes = nil
			}
		}
	} else {
		nextbytes = nil
	}
	return nextbytes
}

func (ioRW *BhIORW) cursorRead(p []byte, ioRWCur *BhReadWriteCursor) (n int, err error) {
	ioRWCur.lockCur()
	defer ioRWCur.unLockCur()
	if pl := len(p); pl > 0 {
		for n < pl {
			if ioRWCur.bytes == nil {
				if ioRWCur.bytes = ioRW.cursorNextReadBytes(ioRWCur); ioRWCur.bytes == nil {
					if n == 0 && err == nil {
						err = io.EOF
					}
					break
				} else {
					ioRWCur.bytesi = 0
				}
			}
			if (pl - n) >= (len(ioRWCur.bytes) - ioRWCur.bytesi) {
				copy(p[n:n+(len(ioRWCur.bytes)-ioRWCur.bytesi)], ioRWCur.bytes[ioRWCur.bytesi:ioRWCur.bytesi+(len(ioRWCur.bytes)-ioRWCur.bytesi)])
				n += (len(ioRWCur.bytes) - ioRWCur.bytesi)
				ioRWCur.seekoffindex += int64((len(ioRWCur.bytes) - ioRWCur.bytesi))
				ioRWCur.bytesi += (len(ioRWCur.bytes) - ioRWCur.bytesi)
			} else if (pl - n) < (len(ioRWCur.bytes) - ioRWCur.bytesi) {
				copy(p[n:n+(pl-n)], ioRWCur.bytes[ioRWCur.bytesi:ioRWCur.bytesi+(pl-n)])
				ioRWCur.bytesi += (pl - n)
				ioRWCur.seekoffindex += int64((pl - n))
				n += (pl - n)
			}
			if len(ioRWCur.bytes) == ioRWCur.bytesi {
				ioRWCur.bytes = nil
			}
		}
	}
	return n, err
}

//Empty -> indicate if BhIORW is empty
func (ioRW *BhIORW) Empty() bool {
	if len(ioRW.buffer) == 0 {
		return ioRW.bytesi == 0
	}
	return false
}

func (ioRW *BhIORW) cursorSeek(offset int64, whence int, ioRWCur *BhReadWriteCursor) (n int64, err error) {
	maxi := ioRW.Size()
	if whence == io.SeekEnd {
		n = maxi
		n -= (offset)
	} else if whence == io.SeekStart {
		n = offset
	} else if whence == io.SeekCurrent {
		n = ioRWCur.seekoffindex
		n += offset
	}
	if n < 0 || n > maxi {
		err = fmt.Errorf("Invalid OffSet IORW must be between %d - %d", 0, maxi)
	} else {
		ioRWCur.seekoffindex = n
		if ioRWCur.bufReadSeeker != nil {
			if ioRWCur.maxBuffer != nil {
				ioRWCur.maxBuffer.Close()
				ioRWCur.maxBuffer = nil
			}
			if ioRWCur.bufReadSeeker != nil {
				if bufsn, bufserr := ioRWCur.bufReadSeeker.Seek(offset, whence); bufserr == nil {
					if bufsn == ioRWCur.seekoffindex {
						ioRWCur.maxBufferSeekIndex = ioRWCur.seekoffindex
					}
				} else {
					err = bufserr
				}
			}
		} else {
			if ioRWCur.bytes != nil {
				ioRWCur.bytes = nil
			}
			ioRWCur.bytesi = 0
		}

	}
	return n, err
}

//BhReadWriteCursor cursor for IORW
type BhReadWriteCursor struct {
	ioRW                   *BhIORW
	bytes                  []byte
	bytesl                 int
	bytesi                 int
	lock                   *sync.Mutex
	isLocked               bool
	seekoffindex           int64
	maxBufferSize          int64
	maxBufferSeekIndex     int64
	maxBufferLastSeekIndex int64
	maxBuffer              *BhIORW
	bufReadSeeker          io.ReadSeeker
	bufReadCloser          io.ReadCloser
}

func (ioRWCur *BhReadWriteCursor) SeekIndex() int64 {
	return ioRWCur.seekoffindex
}

//FileInfo fileinfo
func (ioRWCur *BhReadWriteCursor) FileInfo() os.FileInfo {
	return ioRWCur.ioRW.FileInfo()
}

//ReadAllToHandle perform same action as ReaadAll just calling a custom handle as an ouput (write) caller
func (ioRWCur *BhReadWriteCursor) ReadAllToHandle(hndle func([]byte) (int, error)) (err error) {
	if hndle == nil {
		err = fmt.Errorf("No callable handle assigend")
	} else {
		buf := make([]byte, 4096)
		for {
			if nb, nberr := ioRWCur.Read(buf); nberr == nil || nberr == io.EOF {
				if nb > 0 {
					bi := 0
					for bi < nb {
						if wn, werr := hndle(buf[bi : bi+(nb-bi)]); werr == nil {
							if wn > 0 {
								bi += wn
							}
						} else {
							nberr = werr
							break
						}
					}
				} else {
					if nberr == nil {
						break
					}
				}
				if nberr == io.EOF || nberr != nil {
					err = nberr
					break
				}
			} else {
				err = nberr
				break
			}
		}
	}
	return err
}

//String
func (ioRWCur *BhReadWriteCursor) String() (s string) {
	buf := make([]byte, 4096)
	for {
		if nb, nberr := ioRWCur.Read(buf); nberr == nil || nberr == io.EOF {
			if nb > 0 {
				s += string(buf[0:nb])
			}
			if nberr == io.EOF {
				break
			}
		} else {
			break
		}
	}
	return s
}

//ReadAll content from BhReaderWriterCursor into w io.Writer
func (ioRWCur *BhReadWriteCursor) ReadAll(w io.Writer) (err error) {
	buf := make([]byte, 4096)
	for {
		if nb, nberr := ioRWCur.Read(buf); nberr == nil || nberr == io.EOF {
			if nb > 0 {
				bi := 0
				for bi < nb {
					if wn, werr := w.Write(buf[bi : bi+(nb-bi)]); werr == nil {
						if wn > 0 {
							bi += wn
						}
					} else {
						nberr = werr
						break
					}
				}
			}
			if nberr == io.EOF || nberr != nil {
				err = nberr
				break
			}
		} else {
			err = nberr
			break
		}
	}
	return err
}

func (ioRWCur *BhReadWriteCursor) unLockCur() {
	if ioRWCur.lock != nil {
		if ioRWCur.isLocked {
			ioRWCur.isLocked = false
			ioRWCur.lock.Unlock()
		}
	}
}

func (ioRWCur *BhReadWriteCursor) lockCur() {
	if ioRWCur.lock != nil {
		if !ioRWCur.isLocked {
			ioRWCur.lock.Lock()
			ioRWCur.isLocked = true
		}
	}
}

func (ioRWCur *BhReadWriteCursor) Read(p []byte) (n int, err error) {
	n, err = ioRWCur.ioRW.cursorRead(p, ioRWCur)
	return n, err
}

//Seek -> refer to io.Seeker golang docs
//offset to seek from
//whence -> 0 == From Start
//whence -> 1 == From Current Offset index
//whence -> 2 == From End
func (ioRWCur *BhReadWriteCursor) Seek(offset int64, whence int) (n int64, err error) {
	n, err = ioRWCur.ioRW.cursorSeek(offset, whence, ioRWCur)
	return n, err
}

//Close refer to io.ReaderClose in golang docs
func (ioRWCur *BhReadWriteCursor) Close() (err error) {
	if ioRWCur.lock != nil {
		if ioRWCur.isLocked {
			ioRWCur.unLockCur()
		}
		ioRWCur.lock = nil
	}
	if ioRWCur.seekoffindex > 0 {
		ioRWCur.seekoffindex = 0
	}
	ioRWCur.bytesi = 0
	if ioRWCur.bytes != nil {
		ioRWCur.bytes = nil
	}
	if ioRWCur.maxBuffer != nil {
		ioRWCur.maxBuffer.Close()
		ioRWCur.maxBuffer = nil
	}
	if ioRWCur.bufReadSeeker != nil {
		ioRWCur.bufReadSeeker = nil
	}
	if ioRWCur.bufReadCloser != nil {
		ioRWCur.bufReadCloser.Close()
		ioRWCur.bufReadCloser = nil
	}
	return err
}
