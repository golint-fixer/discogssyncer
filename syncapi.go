package main

import (
	"flag"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/brotherlogic/goserver"
	"github.com/brotherlogic/keystore/client"
	"google.golang.org/grpc"

	pb "github.com/brotherlogic/discogssyncer/server"
	pbd "github.com/brotherlogic/godiscogs"
	pbgs "github.com/brotherlogic/goserver/proto"
)

// Syncer the configuration for the syncer
type Syncer struct {
	*goserver.GoServer
	token       string
	retr        saver
	collection  *pb.RecordCollection
	rMap        map[int]*pbd.Release
	mMap        map[int32]*pb.ReleaseMetadata
	recacheList map[int]*pbd.Release
	mapM        *sync.Mutex
	lastResync  time.Time
}

var (
	syncTime int64
)

const (
	//KEY under which we store the collection
	KEY = "/github.com/brotherlogic/discogssyncer/collection"

	//TOKEN for discogs
	TOKEN = "/github.com/brotherlogic/discogssyncer/token"
)

// This is the only method that interacts with disk
func (s *Syncer) readRecordCollection() error {
	collection := &pb.RecordCollection{}
	data, _, err := s.KSclient.Read(KEY, collection)

	if err != nil {
		return err
	}

	s.collection = data.(*pb.RecordCollection)

	//Ensure we don't keep metadata for release id = 0
	removed := 0
	for i, m := range s.collection.GetMetadata() {
		if m.Id == 0 {
			s.collection.Metadata = append(s.collection.Metadata[:(i-removed)], s.collection.Metadata[(i-removed)+1:]...)
		}
	}

	// Build out the release map
	for _, f := range s.collection.Folders {
		for _, r := range f.Releases.Releases {
			s.rMap[int(r.Id)] = r
		}
	}

	return nil
}

func (s *Syncer) recacheLoop() {
	for true {
		time.Sleep(time.Minute)
		s.resync()
	}
}

func (s *Syncer) saveCollection() {
	t := time.Now()
	s.KSclient.Save(KEY, s.collection)
	s.LogFunction("saveCollection", t)
}

func (s *Syncer) deleteRelease(rel *pbd.Release, folder int32) {
	index := -1
	for _, f := range s.collection.Folders {
		if f.Folder.Id == folder {
			for i, r := range f.Releases.Releases {
				if r.Id == rel.Id {
					index = i
				}

			}
			if index >= 0 {
				f.Releases.Releases = append(f.Releases.Releases[:index], f.Releases.Releases[index+1:]...)
			}
		}
	}

}

// DoRegister does RPC registration
func (s *Syncer) DoRegister(server *grpc.Server) {
	pb.RegisterDiscogsServiceServer(server, s)
}

// InitServer builds an initial server
func InitServer() *Syncer {
	syncer := &Syncer{GoServer: &goserver.GoServer{}, collection: &pb.RecordCollection{Wantlist: &pb.Wantlist{}}, rMap: make(map[int]*pbd.Release), mMap: make(map[int32]*pb.ReleaseMetadata), recacheList: make(map[int]*pbd.Release), lastResync: time.Now()}
	syncer.PrepServer()
	syncer.GoServer.KSclient = *keystoreclient.GetClient(syncer.GetIP)
	syncer.mapM = &sync.Mutex{}

	return syncer
}

// Mote promotes/demotes this server
func (s Syncer) Mote(master bool) error {
	err := s.readRecordCollection()
	return err
}

// GetState gets the state of the server
func (s *Syncer) GetState() []*pbgs.State {
	return []*pbgs.State{&pbgs.State{Key: "last_recache", TimeValue: s.lastResync.Unix()}}
}

// ReportHealth alerts if we're not healthy
func (s Syncer) ReportHealth() bool {
	return true
}

func main() {
	var quiet = flag.Bool("quiet", true, "Show all output")
	var token = flag.String("token", "", "Discogs token")
	flag.Parse()

	//Turn off logging
	if *quiet {
		log.SetFlags(0)
		log.SetOutput(ioutil.Discard)
	}

	syncer := InitServer()

	if len(*token) > 0 {
		syncer.KSclient.Save(TOKEN, &pb.Token{Token: *token})
	}

	tType := &pb.Token{}
	tResp, _, err := syncer.KSclient.Read(TOKEN, tType)

	if err != nil {
		log.Fatalf("Unable to read token: %v", err)
	}

	sToken := tResp.(*pb.Token).Token
	syncer.retr = pbd.NewDiscogsRetriever(sToken)
	syncer.token = sToken
	syncer.RegisterServingTask(syncer.recacheLoop)

	syncer.Register = syncer
	syncer.RegisterServer("discogssyncer", false)

	syncer.Serve()
}
