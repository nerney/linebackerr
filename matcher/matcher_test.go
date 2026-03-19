package matcher

import (
	"errors"
	"testing"
)

func TestMatchCandidateFields(t *testing.T) {
	candidate := MatchCandidate{
		OriginalInput: "NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv",
		GameType:      GameTypeRegularSeason,
		GameDate:      "2021-09-19",
		GameWeek:      "2",
		SeasonYear:    "2021",
		AwayTeam:      "NE",
		HomeTeam:      "NYJ",
	}

	if candidate.OriginalInput != "NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv" {
		t.Fatalf("expected original input NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv, got %q", candidate.OriginalInput)
	}
	if candidate.GameType != GameTypeRegularSeason {
		t.Fatalf("expected game type %q, got %q", GameTypeRegularSeason, candidate.GameType)
	}
	if candidate.GameDate != "2021-09-19" {
		t.Fatalf("expected game date 2021-09-19, got %q", candidate.GameDate)
	}
	if candidate.GameWeek != "2" {
		t.Fatalf("expected game week 2, got %q", candidate.GameWeek)
	}
	if candidate.SeasonYear != "2021" {
		t.Fatalf("expected season year 2021, got %q", candidate.SeasonYear)
	}
	if candidate.AwayTeam != "NE" {
		t.Fatalf("expected away team NE, got %q", candidate.AwayTeam)
	}
	if candidate.HomeTeam != "NYJ" {
		t.Fatalf("expected home team NYJ, got %q", candidate.HomeTeam)
	}
}

func TestMatchFields(t *testing.T) {
	matchErr := errors.New("no nflverse match")
	match := Match{
		OriginalInput: "NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv",
		GameType:      GameTypeRegularSeason,
		GameDate:      "2021-09-19",
		GameWeek:      "2",
		SeasonYear:    "2021",
		AwayTeam:      "NE",
		HomeTeam:      "NYJ",
		NflverseID:    "2021_02_NE_NYJ",
		Error:         matchErr,
	}

	if match.OriginalInput != "NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv" {
		t.Fatalf("expected original input NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv, got %q", match.OriginalInput)
	}
	if match.GameType != GameTypeRegularSeason {
		t.Fatalf("expected game type %q, got %q", GameTypeRegularSeason, match.GameType)
	}
	if match.GameDate != "2021-09-19" {
		t.Fatalf("expected game date 2021-09-19, got %q", match.GameDate)
	}
	if match.GameWeek != "2" {
		t.Fatalf("expected game week 2, got %q", match.GameWeek)
	}
	if match.SeasonYear != "2021" {
		t.Fatalf("expected season year 2021, got %q", match.SeasonYear)
	}
	if match.AwayTeam != "NE" {
		t.Fatalf("expected away team NE, got %q", match.AwayTeam)
	}
	if match.HomeTeam != "NYJ" {
		t.Fatalf("expected home team NYJ, got %q", match.HomeTeam)
	}
	if match.NflverseID != "2021_02_NE_NYJ" {
		t.Fatalf("expected nflverse ID 2021_02_NE_NYJ, got %q", match.NflverseID)
	}
	if !errors.Is(match.Error, matchErr) {
		t.Fatalf("expected match error %v, got %v", matchErr, match.Error)
	}
}

func TestParseReleases(t *testing.T) {
	var files = []string{
		"NFL.2005.Divisional.S2005E019.NE.at.DEN.mkv",
		"NFL.2015.Championship.S2015E021.NE.at.DEN.mkv",
		"NFL.2014.Championship.S2014E020.IND.at.NE.mkv",
		"NFL.2017.Championship.S2017E020.JAX.at.NE.mkv",
		"NFL.2017.Divisional.S2017E019.TEN.at.NE.mkv",
		"NFL.2011.Championship.S2011E020.BAL.at.NE.mkv",
		"NFL.2006.Championship.S2006E020.NE.at.IND.mkv",
		"NFL.2015.Divisional.S2015E019.KC.at.NE.mkv",
		"NFL.2018.Divisional.S2018E019.LAC.at.NE.mkv",
		"NFL.2011.Divisional.S2011E019.DEN.at.NE.mkv",
		"NFL.2018.Championship.S2018E020.NE.at.KC.mkv",
		"NFL.2010.Divisional.S2010E019.NYJ.at.NE.mkv",
		"NFL.2001.Divisional.S2001E019.OAK.at.NE.mkv",
		"NFL.2013.Championship.S2013E020.NE.at.DEN.mkv",
		"NFL.2014.Divisional.S2014E019.BAL.at.NE.mkv",
		"NFL.2006.Divisional.S2006E019.NE.at.SD.mkv",
		"NFL.2013.Divisional.S2013E019.IND.at.NE.mkv",
		"NFL.2006.Wildcard.S2006E018.NYJ.at.NE.mkv",
		"NFL.2016.Divisional.S2016E019.HOU.at.NE.mkv",
		"NFL.2019.Wildcard.S2019E18.NE.at.TEN.mkv",
		"NFL.2012.Divisional.S2012E019.HOU.at.NE.mkv",
		"NFL.2003.Divisional.S2003E019.TEN.at.NE.mkv",
		"NFL.2007.Divisional.S2007E019.JAX.at.NE.mkv",
		"NFL.2025.Divisional.S2025E020.HOU.at.NE.mkv",
		"NFL.2001.Super.Bowl.S2001E021.STL.at.NE.mkv",
		"NFL.2014.Super.Bowl.S2014E021.NE.at.SEA.mkv",
		"NFL.2007.Super.Bowl.S2007E021.NYG.at.NE.mkv",
		"NFL.2003.Super.Bowl.S2003E021.CAR.at.NE.mkv",
		"NFL.2018.Super.Bowl.S2018E021.NE.at.LA.mkv",
		"NFL.2004.Super.Bowl.S2004E021.NE.at.PHI.mkv",
		"NFL.2016.Super.Bowl.S2016E021.NE.at.ATL.mkv",
		"NFL.2017.Super.Bowl.S2017E021.PHI.at.NE.mkv",
		"NFL.2011.Super.Bowl.S2011E021.NYG.at.NE.mkv",
		"NFL.2014-11-16.S2014E011.NE.at.IND.mkv",
		"NFL.2002-11-10.S2002E010.NE.at.CHI.mkv",
		"NFL.2020-12-13.S2020.E13.mkv",
		"NFL.2018.12.16.Patriots.vs.Steelers.S2018E13.WEB-DL.AAC2.0.H.264-720pier.mkv",
		"NFL.2009-10-04.S2009E004.BAL.at.NE.mkv",
		"NFL.2020-12-20.S2020.E15.mkv",
		"NFL.2001-11-11.S2001E009.BUF.at.NE.mkv",
		"NFL.2013-10-20.S2013E007.NE.at.NYJ.mkv",
		"NFL.2001-12-22.S2001E015.MIA.at.NE.mkv",
		"NFL.2005-01-02.S2004E017.SF.at.NE.mkv",
		"NFL.2005-10-16.S2005E006.NE.at.DEN.mkv",
		"NFL.2005-12-17.S2005E015.TB.at.NE.mkv",
		"NFL.2014-11-30.S2014E013.NE.at.GB.mkv",
		"NFL.2012-10-14.S2012E006.NE.at.SEA.mkv",
		"NFL.2011-12-04.S2011E013.IND.at.NE.mkv",
		"NFL.2019-10-27.S2019.E6.Patriots.at.Browns.mkv",
		"NFL.2020-12-26.S2020.E16.mkv",
		"NFL.2020-11-29.S2020.E12.mkv",
		"NFL.2020-11-23.S2020.E11.mkv",
		"NFL.2014-09-14.S2014E002.NE.at.MIN.mkv",
		"NFL.2013-09-29.S2013E004.NE.at.ATL.mkv",
		"NFL.2013-09-22.S2013E003.TB.at.NE.mkv",
		"NFL.2013-10-27.S2013E008.MIA.at.NE.mkv",
		"NFL.2002-11-24.S2002E012.MIN.at.NE.mkv",
		"NFL.2021-09-19.S2021E002.NE.at.NYJ.mkv",
		"NFL.2011-11-06.S2011E009.NYG.at.NE.mkv",
		"NFL.2000-09-03.S2000E001.TB.at.NE.mkv",
		"NFL.2007-09-16.S2007E002.SD.at.NE.mkv",
		"NFL.2014-10-16.S2014E007.NYJ.at.NE.mkv",
		"NFL.2013-11-24.S2013E012.DEN.at.NE.mkv",
		"NFL.2011-12-11.S2011E014.NE.at.WAS.mkv",
		"NFL.2012-12-30.S2012E017.MIA.at.NE.mkv",
		"NFL.2011-10-16.S2011E006.DAL.at.NE.mkv",
		"NFL.2015-11-29.S2015E012.NE.at.DEN.mkv",
		"NFL.2006-12-03.S2006E013.DET.at.NE.mkv",
		"NFL.2014-12-28.S2014E017.BUF.at.NE.mkv",
		"NFL.2019-11-17.S2019E7.NE.at.PHI.mkv",
		"NFL.S2017E2.mkv",
		"NFL.2005-10-30.S2005E008.BUF.at.NE.mkv",
		"NFL.2003-09-28.S2003E004.NE.at.WAS.mkv",
		"NFL.2019-10-21.S2019.E5.Jets.at.Patriots.mkv",
		"NFL.S2017E5.mkv",
		"NFL.2014-12-07.S2014E014.NE.at.SD.mkv",
		"NFL.2001-11-04.S2001E008.NE.at.ATL.mkv",
		"NFL.2005-11-20.S2005E011.NO.at.NE.mkv",
		"NFL.2013-11-18.S2013E011.NE.at.CAR.mkv",
		"NFL.2007-11-04.S2007E009.NE.at.IND.mkv",
		"NFL.2003-10-19.S2003E007.NE.at.MIA.mkv",
		"NFL.2018.12.02.Vikings.vs.Patriots.S2018E11.WEB-DL.AAC2.0.H.264-720pier.mkv",
		"NFL.2004-12-20.S2004E015.NE.at.MIA.mkv",
		"NFL.2020-09-27.S2020.E3.mkv",
		"NFL.2021-09-26.S2021E003.NO.at.NE.mkv",
		"NFL.2014-10-26.S2014E008.CHI.at.NE.mkv",
		"NFL.2012-09-16.S2012E002.ARI.at.NE.mkv",
		"NFL.S2017E1.mkv",
		"NFL.2015-11-08.S2015E009.WAS.at.NE.mkv",
		"NFL.2006-11-26.S2006E012.CHI.at.NE.mkv",
		"NFL.2008-12-28.S2008E017.NE.at.BUF.mkv",
		"NFL.2007-12-03.S2007E013.NE.at.BAL.mkv",
		"NFL.2000-10-08.S2000E006.IND.at.NE.mkv",
		"NFL.2009-11-08.S2009E009.MIA.at.NE.mkv",
		"NFL.2001-12-09.S2001E013.CLE.at.NE.mkv",
		"NFL.2019-11-24.S2019.E8.Cowboys.at.Pats.mkv",
		"NFL.2010-10-04.S2010E004.NE.at.MIA.mkv",
		"NFL.2010-10-24.S2010E007.NE.at.SD.mkv",
		"NFL.2004-11-28.S2004E012.BAL.at.NE.mkv",
		"NFL.2020-09-13.S2020.E1.mkv",
		"NFL.2015-10-25.S2015E007.NYJ.at.NE.mkv",
		"NFL.2015-12-06.S2015E013.PHI.at.NE.mkv",
		"NFL.2011-11-27.S2011E012.NE.at.PHI.mkv",
		"NFL.2015-12-27.S2015E016.NE.at.NYJ.mkv",
		"NFL.2015-12-20.S2015E015.TEN.at.NE.mkv",
		"NFL.2014-12-14.S2014E015.MIA.at.NE.mkv",
		"NFL.2020-10-04.S2020.E4.mkv",
		"NFL.2020-09-20.S2020.E2.mkv",
		"NFL.2020-12-20.S2020.E14.mkv",
		"NFL.2016-11-20.S2016E011.NE.at.SF.mkv",
		"NFL.2020-11-15.S2020.E9.mkv",
		"NFL.S2017E6.mkv",
		"NFL.2019-10-10.S2019.E4.Patriots.at.Giants.mkv",
		"NFL.2014-12-21.S2014E016.NE.at.NYJ.mkv",
		"NFL.2009-10-25.S2009E007.NE.at.TB.mkv",
		"NFL.S2017E8.mkv",
		"NFL.2016-10-09.S2016E005.NE.at.CLE.mkv",
		"NFL.2012-09-09.S2012E001.NE.at.TEN.mkv",
		"NFL.2008-09-21.S2008E003.MIA.at.NE.mkv",
		"NFL.S2017E9.mkv",
		"NFL.2014-11-02.S2014E009.DEN.at.NE.mkv",
		"NFL.2016-11-13.S2016E010.SEA.at.NE.mkv",
		"NFL.2002-10-13.S2002E006.GB.at.NE.mkv",
		"NFL.2013-12-08.S2013E014.CLE.at.NE.mkv",
		"NFL.2003-12-14.S2003E015.JAX.at.NE.mkv",
		"NFL.2013-10-06.S2013E005.NE.at.CIN.mkv",
		"NFL.2007-10-14.S2007E006.NE.at.DAL.mkv",
		"NFL.2014-10-12.S2014E006.NE.at.BUF.mkv",
		"NFL.2016-09-11.S2016E001.NE.at.ARI.mkv",
		"NFL.2007-09-09.S2007E001.NE.at.NYJ.mkv",
		"NFL.2015-09-10.S2015E001.PIT.at.NE.mkv",
		"NFL.2016-10-16.S2016E006.CIN.at.NE.mkv",
		"NFL.2001-10-14.S2001E005.SD.at.NE.mkv",
		"NFL.2003-09-07.S2003E001.NE.at.BUF.mkv",
		"NFL.2012-10-07.S2012E005.DEN.at.NE.mkv",
		"NFL.2017-10-22.S2017E007.ATL.at.NE.mkv",
		"NFL.2008-11-09.S2008E010.BUF.at.NE.mkv",
		"NFL.2019-09-15.S2019E1.MIA.at.NE.mkv",
		"NFL.2000-12-04.S2000E014.KC.at.NE.mkv",
		"NFL.2008-11-30.S2008E013.PIT.at.NE.mkv",
		"NFL.2016-09-22.S2016E003.HOU.at.NE.mkv",
		"NFL.2010-11-14.S2010E010.NE.at.PIT.mkv",
		"NFL.2010-09-12.S2010E001.CIN.at.NE.mkv",
		"NFL.S2017E4.mkv",
		"NFL.2020-10-25.S2020.E7.mkv",
		"NFL.2005-10-02.S2005E004.SD.at.NE.mkv",
		"NFL.2003-09-14.S2003E002.NE.at.PHI.mkv",
		"NFL.2004-10-31.S2004E008.NE.at.PIT.mkv",
		"NFL.2002-09-22.S2002E003.KC.at.NE.mkv",
		"NFL.2015-10-29.S2015E008.MIA.at.NE.mkv",
		"NFL.2000-09-11.S2000E002.NE.at.NYJ.mkv",
		"NFL.2018.12.09.Patriots.vs.Dolphins.S2018E12.WEB-DL.AAC2.0.H.264-720pier.mkv",
		"NFL.2002-09-15.S2002E002.NE.at.NYJ.mkv",
		"NFL.2020-11-03.S2020.E8.mkv",
		"NFL.2007-10-21.S2007E007.NE.at.MIA.mkv",
		"NFL.2008-10-20.S2008E007.DEN.at.NE.mkv",
		"NFL.2019-10-06.S2019.E3.Redskins.at.Patriots.mkv",
		"NFL.2003-11-16.S2003E011.DAL.at.NE.mkv",
		"NFL.2015-12-13.S2015E014.NE.at.HOU.mkv",
		"NFL.2020-10-09.S2020.E5.mkv",
		"NFL.2005-09-18.S2005E002.NE.at.CAR.mkv",
		"NFL.2016-11-27.S2016E012.NE.at.NYJ.mkv",
		"NFL.2015-11-15.S2015E010.NE.at.NYG.mkv",
		"NFL.2021-11-14.S2021E010.CLE.at.NE.mkv",
		"NFL.2011-12-18.S2011E015.NE.at.DEN.mkv",
		"NFL.2011-11-13.S2011E010.NE.at.NYJ.mkv",
		"NFL.2012-10-21.S2012E007.NYJ.at.NE.mkv",
		"NFL.2016-10-02.S2016E004.BUF.at.NE.mkv",
		"NFL.2014-09-07.S2014E001.NE.at.MIA.mkv",
		"NFL.2019-09-29.S2019E2.NE.at.BUF.mkv",
		"NFL.2016-12-24.S2016E016.NYJ.at.NE.mkv",
		"NFL.2012-12-16.S2012E015.SF.at.NE.mkv",
		"NFL.S2017E3.mkv",
		"NFL.2020-11-15.S2020.E10.mkv",
		"NFL.2019-12-08.S2019.E10.Patriots.at.Chiefs.mkv",
		"NFL.2019-12-01.S2019.E9.Texans.at.Patriots.mkv",
		"NFL.2020-10-18.S2020.E6.mkv",
		"NFL.2019-12-21.S2019.E11.Patriots.at.Bills.mkv",
		"NFL.2011-09-25.S2011E003.NE.at.BUF.mkv",
		"NFL.2005-11-13.S2005E010.NE.at.MIA.mkv",
		"NFL.2009.Wildcard.S2009E018.BAL.at.NE.mp4",
		"NFL.2001.Championship.S2001E020.NE.at.PIT.mp4",
		"NFL.2016.Championship.S2016E020.PIT.at.NE.mp4",
		"NFL.2007.Super.Bowl.S2007E021.NYG.at.NE.mp4",
		"NFL.2009-09-27.S2009E003.ATL.at.NE.mp4",
		"NFL.2003-12-07.S2003E014.MIA.at.NE.mp4",
		"NFL.2006-10-08.S2006E005.MIA.at.NE.mp4",
		"NFL.2007-10-21.S2007E007.NE.at.MIA.mp4",
		"NFL.2007-09-23.S2007E003.BUF.at.NE.mp4",
		"NFL.2018.10.14.New.England.Patriots.vs.Kansas.City.Chiefs.S2018E6.HDTV.AAC2.0.x264.mp4",
		"NFL.2018-10-21.S2018E007.NE.at.CHI.mp4",
		"NFL.2008-09-21.S2008E003.MIA.at.NE.mp4",
		"NFL.2009-12-13.S2009E014.CAR.at.NE.mp4",
		"NFL.2018.11.25.New.York.Jets.vs.New.England.Patriots.S2018E10.HDTV.AAC2.0.x264.mp4",
		"NFL.2000-10-01.S2000E005.NE.at.DEN.mp4",
		"NFL.2010-11-14.S2010E010.NE.at.PIT.mp4",
		"NFL.2012-09-23.S2012E003.NE.at.BAL.mp4",
		"NFL.2009-11-15.S2009E010.NE.at.IND.mp4",
		"NFL.2016-10-23.S2016E007.NE.at.PIT.mp4",
		"NFL.2010-12-12.S2010E014.NE.at.CHI.mp4",
		"NFL.S2017E19.mp4",
		"NFL.2018.10.04.New.England.Patriots.vs.Indianapolis.Colts.S2018E5.HDTV.AAC2.0.x264.mp4",
		"NFL.2002-01-06.S2001E017.NE.at.CAR.mp4",
		"NFL.2008-12-21.S2008E016.ARI.at.NE.mp4",
		"NFL.2008-09-07.S2008E001.KC.at.NE.mp4",
		"NFL.2018.11.04.New.England.Patriots.vs.Green.Bay.Packers.S2018E9.HDTV.AAC2.0.H264.mp4",
		"NFL.2007-10-14.S2007E006.NE.at.DAL.mp4",
		"NFL.2018.09.09.New.England.Patriots.vs.Houston.Texans.S2018E3.HDTV.AAC2.0.x264.mp4",
		"NFL.2009-09-14.S2009E001.BUF.at.NE.mp4",
		"NFL.2018.09.23.Detroit.Lions.vs.New.England.Patriots.S2018E4.HDTV.AAC2.0.x264.mp4",
		"NFL.2007-12-03.S2007E013.NE.at.BAL.mp4",
		"NFL.S2017E0.mp4",
		"NFL.2007-10-01.S2007E004.NE.at.CIN.mp4",
		"NFL.2003-11-23.S2003E012.NE.at.HOU.mp4",
		"NFL.2006-11-05.S2006E009.IND.at.NE.mp4",
		"NFL.1999-10-24.S1999E007.DEN.at.NE.mp4",
		"NFL.2018.10.29.Buffalo.Bills.vs.New.England.Patriots.S2018E8.HDTV.AAC2.0.x264.mp4",
		"NFL.2008-12-07.S2008E014.NE.at.SEA.mp4",
		"NFL.2009-11-08.S2009E009.MIA.at.NE.mp4",
		"NFL.2007-12-29.S2007E017.NE.at.NYG.mp4",
		"NFL.2007-11-18.S2007E011.NE.at.BUF.mp4",
		"NFL.2004-10-31.S2004E008.NE.at.PIT.mp4",
		"NFL.2016-12-18.S2016E015.NE.at.DEN.mp4",
		"NFL.2007-10-07.S2007E005.CLE.at.NE.mp4",
	}

	results := ParseReleases(files)
	if len(results) != len(files) {
		t.Errorf("Expected %d results, got %d", len(files), len(results))
	}
	// Print outputs for manual verification or add specific assertions
	for i, res := range results {
		t.Logf("In:  %s\nOut: %s\n", files[i], res)
	}
}
