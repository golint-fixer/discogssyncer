package main

import (
	"log"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"

	pb "github.com/brotherlogic/discogssyncer/server"
	pbd "github.com/brotherlogic/godiscogs"
	"github.com/brotherlogic/goserver"
)

type testDiscogsRetriever struct{}

func (testDiscogsRetriever) GetCollection() []pbd.Release {
	var releases = make([]pbd.Release, 0)
	releases = append(releases, pbd.Release{FolderId: 23, Id: 25, MasterId: 234})
	releases = append(releases, pbd.Release{FolderId: 23, Id: 32, MasterId: 245})
	releases = append(releases, pbd.Release{FolderId: 22, Id: 29, MasterId: 234})
	releases = append(releases, pbd.Release{FolderId: 22, Id: 65})
	releases = append(releases, pbd.Release{FolderId: 22, Id: 79})
	return releases
}

func (testDiscogsRetriever) GetRelease(id int) (pbd.Release, error) {
	return pbd.Release{Id: int32(id)}, nil
}

func (testDiscogsRetriever) GetFolders() []pbd.Folder {
	var folders = make([]pbd.Folder, 0)
	folders = append(folders, pbd.Folder{Id: 23, Name: "Testing"})
	folders = append(folders, pbd.Folder{Id: 25, Name: "TestingTwo"})
	return folders
}

func (testDiscogsRetriever) GetWantlist() ([]pbd.Release, error) {
	var wants = make([]pbd.Release, 0)
	wants = append(wants, pbd.Release{FolderId: 23, Id: 256})
	wants = append(wants, pbd.Release{FolderId: 23, Id: 324})
	return wants, nil
}

func (testDiscogsRetriever) MoveToFolder(fodlerID int, releaseID int, instanceID int, newFolderID int) {
	// Do nothing
}

func (testDiscogsRetriever) AddToFolder(fodlerID int, releaseID int) {
	// Do nothing
}

func (testDiscogsRetriever) SetRating(folderID int, releaseID int, instanceID int, rating int) {
	// Do nothing
}

func (testDiscogsRetriever) RemoveFromWantlist(releaseID int) {
	// Do nothing
}

func (testDiscogsRetriever) AddToWantlist(releaseID int) {
	// Do nothing
}

func TestSearch(t *testing.T) {
	syncer := GetTestSyncer(".testSearch", true)
	release1 := &pbd.Release{Title: "Spiderland", FolderId: 23, Id: 25, InstanceId: 37}
	release2 := &pbd.Release{Title: "FutureWorld", FolderId: 23, Id: 27, InstanceId: 39}

	syncer.saveRelease(release1, 23)
	syncer.saveRelease(release2, 23)

	res, err := syncer.Search(context.Background(), &pb.SearchRequest{Query: "spider"})
	if err != nil {
		t.Errorf("Failure to search: %v", err)
	}

	if len(res.Releases) != 1 || res.Releases[0].Id != 25 {
		t.Errorf("Search has failed: %v", res)
	}
}

func TestGetMetadata(t *testing.T) {
	sTime := time.Now().Unix()
	syncer := GetTestSyncer(".testGetMetadata", true)
	release := &pbd.Release{FolderId: 23, Id: 25, InstanceId: 37}
	syncer.saveRelease(release, 23)
	metadata, err := syncer.GetMetadata(context.Background(), release)

	if err != nil || metadata == nil {
		t.Errorf("Error in get metadata : %v", err)
	}

	if metadata != nil {
		if metadata.DateAdded < sTime {
			t.Errorf("metadata was not stored")
		}

		if metadata.Cost < 0 {
			t.Errorf("Cost was not stored")
		}
	}
}

func TestGetMetadataFailWithNoMetadata(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testGetMetadataFailWithNo")
	release := &pbd.Release{FolderId: 23, Id: 25, InstanceId: 37}
	syncer.saveRelease(release, 23)

	//Force delete the metadata
	os.Remove(".testGetMetadataFailWithNo/static-metadata/25.metadata")

	metadata, err := syncer.GetMetadata(context.Background(), release)
	if err == nil {
		t.Errorf("Error in get metadata, should have errored : %v", metadata)
	}
}

func TestGetMonthlySpend(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testGetMetadata")
	release := &pbd.Release{FolderId: 23, Id: 25, InstanceId: 37}
	syncer.AddToFolder(context.Background(), &pb.ReleaseMove{Release: release, NewFolderId: 23})
	metadata, _ := syncer.GetMetadata(context.Background(), release)
	metadata.Cost = 200
	birthday, _ := time.Parse("02/01/06", "22/10/77")
	metadata.DateAdded = birthday.Unix()

	syncer.UpdateMetadata(context.Background(), &pb.MetadataUpdate{Release: release, Update: metadata})

	spend, err := syncer.GetSpend(context.Background(), &pb.SpendRequest{Month: 10, Year: 1977})
	if err != nil {
		t.Errorf("Fail to get monthly spend: %v", err)
	}
	if int(spend.TotalSpend) != 200 {
		t.Errorf("Monthly spend is miscalculated: %v", spend)
	}
}

func TestGetYearlySpend(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testGetYearlySpend")
	release := &pbd.Release{FolderId: 23, Id: 25, InstanceId: 37}
	syncer.AddToFolder(context.Background(), &pb.ReleaseMove{Release: release, NewFolderId: 23})
	metadata, _ := syncer.GetMetadata(context.Background(), release)
	metadata.Cost = 200
	birthday, _ := time.Parse("02/01/06", "22/10/77")
	metadata.DateAdded = birthday.Unix()

	syncer.UpdateMetadata(context.Background(), &pb.MetadataUpdate{Release: release, Update: metadata})

	spend, err := syncer.GetSpend(context.Background(), &pb.SpendRequest{Year: 1977})
	if err != nil {
		t.Errorf("Fail to get yearly spend: %v", err)
	}
	if int(spend.TotalSpend) != 200 {
		t.Errorf("Yearly spend is miscalculated: %v", spend)
	}
}

func TestGetWantlist(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testwantlist")
	syncer.wants.Want = append(syncer.wants.Want, &pb.Want{ReleaseId: 25})
	syncer.SyncWantlist()
	wantlist, err := syncer.GetWantlist(context.Background(), &pb.Empty{})

	if err != nil {
		t.Errorf("Error getting wantlist: %v", err)
	}

	if len(wantlist.Want) == 0 {
		t.Errorf("No wants returned")
	}
}

func TestDeleteWantFully(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testwantlistfully")
	syncer.SyncWantlist()
	wantlist, err := syncer.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error getting wantlist: %v", err)
	}
	log.Printf("WANTS = %v", wantlist)
	if len(wantlist.Want) == 0 {
		t.Errorf("No wants returned")
	}

	syncer.DeleteWant(context.Background(), &pb.Want{ReleaseId: 256})
	wantlist, err = syncer.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error getting wantlist: %v", err)
	}
	if len(wantlist.Want) != 1 {
		t.Errorf("Wants returned post delete: %v", wantlist)
	}

	nsyncer := GetTestSyncerNoDelete(".testwantlistfully")
	wantlist, err = nsyncer.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error getting wantlist: %v", err)
	}
	if len(wantlist.Want) != 1 {
		t.Errorf("Wants returned on reload: %v", wantlist)
	}
}

func TestSetWant(t *testing.T) {
	syncer := GetTestSyncer(".testsetwant", true)
	syncer.wants.Want = append(syncer.wants.Want, &pb.Want{ReleaseId: 256, Wanted: true})

	wantedit := &pb.Want{ReleaseId: 256, Valued: true}
	syncer.EditWant(context.Background(), wantedit)

	wantlist, err := syncer.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error getting wantlist: %v", err)
	}

	if len(wantlist.Want) != 1 {
		t.Errorf("Wrong number of wants : %v", wantlist)
	}

	if !wantlist.Want[0].Valued {
		t.Errorf("Want is not valued: %v", wantlist)
	}

	wantedit.Valued = false
	syncer.EditWant(context.Background(), wantedit)

	wantlist, err = syncer.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error getting wantlist: %v", err)
	}

	if wantlist.Want[0].Valued {
		t.Errorf("Want has remained valued %v", wantlist)
	}
}

func TestSaveWantDoesNotSaveMetadata(t *testing.T) {
	syncer := GetTestSyncer(".testsavewantdoesnotsavemetadata", true)
	r := &pbd.Release{Id: 25, Title: "MadeUpRelease", FolderId: -5}
	syncer.saveRelease(r, -5)

	meta, err := syncer.GetMetadata(context.Background(), r)
	if err != nil {
		t.Errorf("Failure in get metadata: %v", err)
	} else {
		if meta.DateAdded > 0 {
			t.Errorf("Wantlist sync has set the date added: %v", meta)
		}
	}
	log.Printf("META = %v", meta)

	r3 := &pbd.Release{Id: 25, Title: "MadeUpRelease", FolderId: 24}
	syncer.saveRelease(r3, 24)
	meta3, err := syncer.GetMetadata(context.Background(), r3)
	if err != nil {
		t.Errorf("Failure in get metadata: %v", err)
	} else {
		if meta3.DateAdded <= 0 {
			t.Errorf("Want has not converted to purchase: %v", meta3)
		}
	}
}

func TestCollapseWantlist(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testcollapsewants")
	syncer.wants.Want = append(syncer.wants.Want, &pb.Want{ReleaseId: 256, Wanted: true})
	syncer.wants.Want = append(syncer.wants.Want, &pb.Want{ReleaseId: 257, Valued: true, Wanted: true})
	syncer.SyncWantlist()
	wantlist, err := syncer.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error getting wantlist: %v", err)
	}
	if !wantlist.Want[0].Wanted {
		t.Errorf("Initial want is not wanted: %v", wantlist)
	}

	nwantlist, err := syncer.CollapseWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error collapseing wantlist: %v", err)
	}
	if nwantlist.Want[0].Wanted {
		t.Errorf("Want has not been collapsed: %v", nwantlist)
	}
	if !nwantlist.Want[1].Wanted {
		t.Errorf("Valued records as not been maintained: %v", nwantlist)
	}

	nwantlist2, err := syncer.RebuildWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error rebuilding wantlist: %v", err)
	}
	if !nwantlist2.Want[0].Wanted {
		t.Errorf("Want has not been rebuilt: %v", nwantlist2)
	}
	if !nwantlist2.Want[1].Wanted {
		t.Errorf("Valued records as not been maintained: %v (%v)", nwantlist2, nwantlist2.Want[1])
	}
}

func TestDeleteWant(t *testing.T) {
	syncer := GetTestSyncer(".testsetwant", true)
	syncer.wants.Want = append(syncer.wants.Want, &pb.Want{ReleaseId: 256, Wanted: true})

	deleteWant := &pb.Want{ReleaseId: 256}
	syncer.DeleteWant(context.Background(), deleteWant)

	wantlist, err := syncer.GetWantlist(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error getting wantlist: %v", err)
	}
	if len(wantlist.Want) != 0 {
		t.Errorf("Wrong number of wants returned: %v", wantlist)
	}
}

func TestAddToFolder(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testAddToFolder")
	release := &pbd.Release{Id: 25}
	releaseMove := &pb.ReleaseMove{Release: release, NewFolderId: 20}
	_, err := syncer.AddToFolder(context.Background(), releaseMove)
	if err != nil {
		t.Errorf("Move to uncat has returned error")
	}

	newRelease, _ := syncer.GetRelease(25, 20)
	if newRelease == nil || newRelease.FolderId != 20 {
		t.Errorf("Error in retrieving added release: %v", newRelease)
	}
}

func TestGetRelease(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testGetRelease")
	release := &pbd.Release{Id: 25}
	releaseMove := &pb.ReleaseMove{Release: release, NewFolderId: 20}
	_, err := syncer.AddToFolder(context.Background(), releaseMove)
	if err != nil {
		t.Errorf("Move to uncat has returned error")
	}

	newRelease, err := syncer.GetSingleRelease(context.Background(), release)
	if err != nil || newRelease.FolderId != 20 {
		t.Errorf("Error in retrieving added release: %v", newRelease)
	}
}

func TestGetReleaseFromCache(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testGetReleaseCache")
	release := &pbd.Release{Id: 25}
	releaseMove := &pb.ReleaseMove{Release: release, NewFolderId: 20}
	_, err := syncer.AddToFolder(context.Background(), releaseMove)
	if err != nil {
		t.Errorf("Move to uncat has returned error")
	}

	// The first should put us in the cache
	syncer.GetSingleRelease(context.Background(), release)
	newRelease, err := syncer.GetSingleRelease(context.Background(), release)
	if err != nil || newRelease.FolderId != 20 {
		t.Errorf("Error in retrieving added release: %v", newRelease)
	}
}

func TestGetReleaseGetsWantlist(t *testing.T) {
	syncer := GetTestSyncer(".testreleasewant", true)
	release := &pbd.Release{Id: 25}
	releaseMove := &pb.ReleaseMove{Release: release, NewFolderId: 20}
	_, err := syncer.AddToFolder(context.Background(), releaseMove)
	if err != nil {
		t.Errorf("Error in adding release: %v", err)
	}
	syncer.wants.Want = append(syncer.wants.Want, &pb.Want{ReleaseId: 256, Wanted: true})
	syncer.SyncWantlist()

	// Retrieve a want directly
	retRel, err := syncer.GetSingleRelease(context.Background(), &pbd.Release{Id: 256})
	if err != nil {
		t.Errorf("Error in getting release: %v", err)
	}
	if retRel.Id != 256 {
		t.Errorf("Release has come back bad: %v", retRel)
	}
}

func TestGetReleaseFail(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testGetNoRelease")
	release := &pbd.Release{Id: 25}
	newRelease, err := syncer.GetSingleRelease(context.Background(), release)
	if err == nil {
		t.Errorf("Failed to error on release: %v", newRelease)
	}
}

func TestSaveCollection(t *testing.T) {
	syncer := Syncer{saveLocation: ".testcollectionsave/"}
	syncer.SaveCollection(&testDiscogsRetriever{})
}

func TestGetCollection(t *testing.T) {
	syncer := Syncer{saveLocation: ".testcollectionsave/", cache: make(map[int32]string)}
	syncer.SaveCollection(&testDiscogsRetriever{})

	releases, err := syncer.GetCollection(context.Background(), &pb.Empty{})
	if err != nil {
		t.Errorf("Error returned on Get Collection")
	}

	if len(releases.Releases) == 0 {
		t.Errorf("No releases have been returned")
	}

	folders := syncer.getFolders()
	if len(folders.Folders) != 2 {
		t.Errorf("Not enough folders: %v", folders)
	}

	if folders.Folders[0].Name == folders.Folders[1].Name {
		t.Errorf("FOlders have same name: %v", folders)
	}
}

func TestGetCollectionNoWantlist(t *testing.T) {
	syncer := GetTestSyncer(".testcollectionnowantlist", true)
	syncer.wants.Want = append(syncer.wants.Want, &pb.Want{ReleaseId: 56})
	syncer.SyncWantlist()
	syncer.SaveCollection(&testDiscogsRetriever{})

	releases, err := syncer.GetCollection(context.Background(), &pb.Empty{})

	if err != nil {
		t.Errorf("Error returned on Get Collection %v", err)
	}

	for _, rel := range releases.Releases {
		if rel.FolderId == -1 || rel.FolderId == 0 {
			t.Errorf("GetCollection has returned something on the wantlist: %v", rel)
		} else {
			log.Printf("SHERE = %v", rel)
		}
	}
}

func TestRetrieveEmptyCollection(t *testing.T) {
	syncer := Syncer{saveLocation: ".testemptyfolder/"}
	folderList := &pb.FolderList{}
	folderList.Folders = append(folderList.Folders, &pbd.Folder{Name: "TestOne", Id: 1234})
	rels, err := syncer.GetReleasesInFolder(context.Background(), folderList)
	if err == nil && len(rels.Releases) > 0 {
		t.Errorf("Pull from empty folder returns no error! or valid releases")
	}
}

func TestSaveLocation(t *testing.T) {
	syncer := Syncer{saveLocation: ".testfolder/"}
	release := &pbd.Release{Id: 1234}
	syncer.saveRelease(release, 12)

	//Check that the file is in the right location
	if _, err := os.Stat(".testfolder/12/1234.release"); os.IsNotExist(err) {
		t.Errorf("File does not exists")
	}
}

func TestSaveMetadata(t *testing.T) {
	now := time.Now()
	syncer := Syncer{saveLocation: ".testmetadatasave/", cache: make(map[int32]string)}
	release := &pbd.Release{Id: 1234}
	syncer.saveRelease(release, 12)

	_, metadata := syncer.GetRelease(1234, 12)
	if metadata.DateAdded > now.Unix() {
		t.Errorf("Metadata is prior to adding the release: %v (%v)", metadata.DateAdded, metadata.DateAdded-now.Unix())
	}
}

func TestUpdateMetadata(t *testing.T) {
	syncer := GetTestSyncer(".testupdatemetadata", true)
	release := &pbd.Release{FolderId: 23, Id: 25, InstanceId: 37}
	syncer.saveRelease(release, 23)

	_, metadata := syncer.GetRelease(25, 23)
	if metadata.DateAdded == 1234 {
		t.Errorf("Test bleed through on metadata: %v", metadata)
	}

	newMetadata := &pb.ReleaseMetadata{DateAdded: 1234}
	retMetadata, err := syncer.UpdateMetadata(context.Background(), &pb.MetadataUpdate{Release: release, Update: newMetadata})
	if err != nil {
		t.Errorf("Failed metadata update: %v", err)
	}

	if retMetadata.DateAdded != 1234 {
		t.Errorf("Date Added has not been updated: %v", retMetadata.DateAdded)
	}

	_, metadataStored := syncer.GetRelease(25, 23)
	if metadataStored.DateAdded != 1234 {
		t.Errorf("Date Added has not been stored: %v", metadataStored.DateAdded)
	}
}

func TestUpdateMetadataFail(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testupdatemetadatafail")
	release := &pbd.Release{FolderId: 23, Id: 25, InstanceId: 37}
	syncer.saveRelease(release, 23)

	_, metadata := syncer.GetRelease(25, 23)
	if metadata.DateAdded == 1234 {
		t.Errorf("Test bleed through on metadata: %v", metadata)
	}

	newMetadata := &pb.ReleaseMetadata{DateAdded: 1234}
	newRelease := &pbd.Release{FolderId: 23, Id: 27, InstanceId: 37}
	_, err := syncer.UpdateMetadata(context.Background(), &pb.MetadataUpdate{Release: newRelease, Update: newMetadata})
	if err == nil {
		t.Errorf("Metadata failed to return error: %v", err)
	}
}

func TestUpdateRating(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testupdaterating")
	release := &pbd.Release{FolderId: 23, Id: 25, InstanceId: 37}
	syncer.saveRelease(release, 23)

	release.Rating = 5
	syncer.UpdateRating(context.Background(), release)

	oldRelease, _ := syncer.GetRelease(25, 23)
	if oldRelease.Rating != 5 {
		t.Errorf("Rating has not been saved %v", oldRelease)
	}
}

func TestSaveAndRefreshMetadata(t *testing.T) {
	now := time.Now()
	syncer := Syncer{saveLocation: ".testmetadatasave/", cache: make(map[int32]string)}
	release := &pbd.Release{Id: 1234}
	syncer.saveRelease(release, 12)

	_, metadata := syncer.GetRelease(1234, 12)
	if metadata.DateAdded > now.Unix() {
		t.Errorf("Metadata is prior to adding the release: %v (%v)", metadata.DateAdded, metadata.DateAdded-now.Unix())
	}

	time.Sleep(time.Second)
	syncer.saveRelease(release, 12)
	_, metadata2 := syncer.GetRelease(1234, 12)
	if metadata2.DateRefreshed == metadata.DateRefreshed {
		t.Errorf("Metadata has not been refreshed: %v and %v", metadata.DateRefreshed, metadata2.DateRefreshed)
	}
}

func GetTestSyncer(foldername string, delete bool) Syncer {
	syncer := Syncer{
		saveLocation: foldername,
		retr:         testDiscogsRetriever{},
		cache:        make(map[int32]string),
	}

	if delete {
		os.RemoveAll(syncer.saveLocation)
	}

	syncer.GoServer = &goserver.GoServer{}
	syncer.SkipLog = true
	syncer.Register = syncer
	syncer.initWantlist()
	return syncer
}

func GetTestSyncerNoDelete(foldername string) Syncer {
	return GetTestSyncer(foldername, false)
}

func TestGetFolderById(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testgetfolders/")
	folders := &pb.FolderList{}
	folders.Folders = append(folders.Folders, &pbd.Folder{Name: "TestOne", Id: 1234})
	syncer.SaveFolders(folders)

	release := &pbd.Release{Id: 1234}
	syncer.saveRelease(release, 1234)

	folderList := &pb.FolderList{}
	folderList.Folders = append(folderList.Folders, &pbd.Folder{Id: 1234})
	releases, err := syncer.GetReleasesInFolder(context.Background(), folderList)
	if err != nil {
		t.Errorf("Failure to get releases: %v", err)
	}
	if len(releases.Releases) != 1 {
		t.Errorf("Bad retrieve of releases: %v", releases)
	}
}

func TestGetFolders(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testgetfolders/")
	folders := &pb.FolderList{}
	folders.Folders = append(folders.Folders, &pbd.Folder{Name: "TestOne", Id: 1234})
	folders.Folders = append(folders.Folders, &pbd.Folder{Name: "TestTwo", Id: 1235})
	syncer.SaveFolders(folders)

	release := &pbd.Release{Id: 1234}
	syncer.saveRelease(release, 1234)

	folderList := &pb.FolderList{}
	folderList.Folders = append(folderList.Folders, &pbd.Folder{Name: "TestOne"})
	releases, err := syncer.GetReleasesInFolder(context.Background(), folderList)

	folderList2 := &pb.FolderList{}
	folderList2.Folders = append(folderList2.Folders, &pbd.Folder{Name: "TestTwo"})
	releases2, _ := syncer.GetReleasesInFolder(context.Background(), folderList2)

	if err != nil {
		t.Errorf("Error retrieveing releases: %v", err)
	}

	if len(releases.Releases) == 0 {
		t.Errorf("GetReleasesInFolder came back empty")
	}

	if len(releases2.Releases) != 0 {
		t.Errorf("Releases returned for folder 2: %v", releases2)
	}
}

func TestSaveFolderMetaata(t *testing.T) {
	syncer := GetTestSyncerNoDelete(".testSaveFolderMetadata/")
	folderList := &pb.FolderList{}
	folderList.Folders = append(folderList.Folders, &pbd.Folder{Name: "TestOne", Id: 1234})
	folderList.Folders = append(folderList.Folders, &pbd.Folder{Name: "TestTwo", Id: 1232})

	syncer.SaveFolders(folderList)

	if _, err := os.Stat(".testSaveFolderMetadata/metadata/folders"); os.IsNotExist(err) {
		t.Errorf("Folder metedata has not been save")
	}
}

func TestOtherCopies(t *testing.T) {
	syncer := GetTestSyncer(".testOtherCopies", true)
	syncer.SaveCollection(&testDiscogsRetriever{})
	syncer.SaveCollection(&testDiscogsRetriever{})

	// Some releases should be marked as having other copies
	relHas, meta := syncer.GetRelease(29, 22)
	log.Printf("META1 = %v", meta)
	if !meta.Others {
		t.Errorf("%v has not been marked as having others: %v", relHas, meta)
	}
	relHasnot, meta := syncer.GetRelease(32, 23)
	log.Printf("META2 = %v", meta)
	if meta.Others {
		t.Errorf("%v has actually been marked as having others: %v", relHasnot, meta)
	}
	relNoParent, meta := syncer.GetRelease(65, 22)
	log.Printf("META3 = %v", meta)
	if meta.Others {
		t.Errorf("%v has actually been marked as having others: %v", relNoParent, meta)
	}
}
