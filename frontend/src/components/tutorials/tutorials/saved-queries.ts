import { Tutorial } from "@/types/tutorial"

export const savedQueriesTutorial: Tutorial = {
  id: "saved-queries",
  name: "Working with Saved Queries",
  description: "Learn how to save, organize, and share your frequently used queries",
  category: "queries",
  difficulty: "beginner",
  estimatedMinutes: 4,
  steps: [
    {
      target: "[data-tutorial='save-query-button']",
      title: "Saving a Query",
      content: `
        <p>After writing a query, click the <strong>Save</strong> button to store it for later use.</p>
        <p>Saved queries are:</p>
        <ul>
          <li>Accessible from the sidebar</li>
          <li>Synced across devices (with cloud sync enabled)</li>
          <li>Shareable with your team</li>
        </ul>
      `,
      placement: "bottom",
      action: {
        type: "click",
        instruction: "Click the Save button",
      },
    },
    {
      target: "[data-tutorial='query-name-input']",
      title: "Naming Your Query",
      content: `
        <p>Give your query a descriptive name so you can find it easily later.</p>
        <p>Good names describe what the query does, like:</p>
        <ul>
          <li>"Active Users Report"</li>
          <li>"Monthly Sales Summary"</li>
          <li>"User Login Analytics"</li>
        </ul>
      `,
      placement: "right",
    },
    {
      target: "[data-tutorial='query-folders']",
      title: "Organizing with Folders",
      content: `
        <p>Create folders to organize your queries by:</p>
        <ul>
          <li>Project</li>
          <li>Database</li>
          <li>Query type (Reports, Analytics, etc.)</li>
          <li>Team or department</li>
        </ul>
        <p>Drag and drop queries between folders to reorganize.</p>
      `,
      placement: "right",
    },
    {
      target: "[data-tutorial='favorite-query']",
      title: "Favoriting Queries",
      content: `
        <p>Click the star icon to mark queries as favorites.</p>
        <p>Favorited queries appear at the top of your list for quick access.</p>
      `,
      placement: "left",
      action: {
        type: "click",
        instruction: "Click the star to favorite this query",
      },
    },
    {
      target: "[data-tutorial='share-query']",
      title: "Sharing with Your Team",
      content: `
        <p>Share queries with team members by clicking the share button.</p>
        <p>You can:</p>
        <ul>
          <li>Share with specific people</li>
          <li>Share with your entire organization</li>
          <li>Set view-only or edit permissions</li>
        </ul>
      `,
      placement: "left",
    },
    {
      target: "[data-tutorial='query-versions']",
      title: "Version History",
      content: `
        <p>Howlerops automatically tracks changes to your saved queries.</p>
        <p>View previous versions and restore them if needed.</p>
        <p>This is especially useful when collaborating with others!</p>
      `,
      placement: "bottom",
    },
  ],
}
