package pages

import (
 "github.com/ramon/trackline/internal/calendar"
 "github.com/ramon/trackline/internal/database"
 "github.com/ramon/trackline/web/layouts"
)

type DashboardData struct {
 Base layouts.Data
 Stats database.DashboardStatsRow
 Recent []database.RecentIssuesRow
 Activity []database.ListRecentActivityRow
 Release database.CurrentReleaseRow
 Groups map[string][]database.ReportGroupsRow
}
type ProjectData struct {
 Base layouts.Data
 Project database.Project
 Tab string
 Issues []database.ListProjectIssuesRow
 Board []database.ListBoardIssuesRow
 Sprints []database.Sprint
 SprintID int64
 Releases []database.ListReleasesRow
 Members []database.ListProjectMembersRow
 Users []database.ListUsersRow
 Repository database.GetGitHubRepositoryByProjectRow
 GitHubEnabled bool
 GitHubSlug string
 Page int
 Status string
 Type string
 HasNext bool
 Calendar calendar.Month
 Reports map[string][]database.ReportGroupsRow
 Estimates []database.ProjectTimeReportRow
 DateFrom string
 DateTo string
}
type IssueData struct {
 Base layouts.Data
 Item database.GetIssueByKeyRow
 Sprints []database.Sprint
 Users []database.ListUsersRow
 Releases []database.ListProjectReleaseOptionsRow
 Labels []database.Label
 SelectedLabels []database.Label
 Comments []database.ListCommentsRow
 Activity []database.ListIssueActivityRow
 Time []database.ListIssueTimeRow
 TimeSummary []database.IssueTimeSummaryRow
 Commits []database.GithubCommit
 PRs []database.GithubPullRequest
 Spent int64
}
type NewIssueData struct {
 Base layouts.Data
 Project database.Project
 Users []database.ListUsersRow
 Releases []database.ListProjectReleaseOptionsRow
 Labels []database.Label
}
type ReleaseData struct {
 Base layouts.Data
 Item database.GetReleaseRow
 Issues []database.ListReleaseIssuesRow
 Counts database.ReleaseGitHubCountsRow
 Spent int64
 GitHubEnabled bool
}
type SettingsData struct {
 Base layouts.Data
 Users []database.ListUsersRow
 Labels []database.Label
 GitHubEnabled bool
}
type SearchData struct {
 Base layouts.Data
 Query string
 Issues []database.SearchIssuesRow
 Others []database.SearchProjectsAndReleasesRow
}
