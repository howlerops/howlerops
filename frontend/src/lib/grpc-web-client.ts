import {
  ConnectionHealthResponse,
  CreateConnectionRequest,
  CreateConnectionResponse,
  DeleteConnectionRequest,
  DeleteConnectionResponse,
  GetConnectionHealthRequest,
  GetConnectionRequest,
  GetConnectionResponse,
  GetSchemasRequest,
  GetSchemasResponse,
  GetTableMetadataRequest,
  GetTableMetadataResponse,
  GetTablesRequest,
  GetTablesResponse,
  ListConnectionsRequest,
  ListConnectionsResponse,
  TestConnectionRequest,
  TestConnectionResponse,
  UpdateConnectionRequest,
  UpdateConnectionResponse,
} from '../generated/database';
import {
  CancelQueryRequest,
  CancelQueryResponse,
  ExecuteBatchEditRequest,
  ExecuteBatchEditResponse,
  ExecuteQueryRequest,
  ExecuteQueryResponse,
  ExecuteStreamingQueryRequest,
  ExplainQueryRequest,
  ExplainQueryResponse,
  GetQueryHistoryRequest,
  GetQueryHistoryResponse,
  GetQueryStatusRequest,
  GetQueryStatusResponse,
  ListActiveQueriesRequest,
  ListActiveQueriesResponse,
  StreamingQueryResponse,
} from '../generated/query';

// Configuration for gRPC-Web client
const HTTP_GATEWAY_ENDPOINT = import.meta.env.VITE_HTTP_GATEWAY_ENDPOINT || 'http://localhost:8500';

// HTTP-based gRPC-Web client implementation
export class GrpcWebClient {
  private baseUrl: string;
  private activeStreams: Map<string, AbortController> = new Map();

  constructor() {
    this.baseUrl = HTTP_GATEWAY_ENDPOINT;
  }

  private async makeRequest<T>(
    servicePath: string,
    request: unknown
  ): Promise<T> {
    const url = `${this.baseUrl}${servicePath}`;

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'application/json',
        },
        body: JSON.stringify(request),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`HTTP ${response.status}: ${errorText}`);
      }

      const data = await response.json();
      return data as T;
    } catch (error) {
      throw new Error(`gRPC request failed: ${error}`);
    }
  }

  private async makeStreamingRequest<T>(
    servicePath: string,
    request: unknown,
    onMessage: (message: T) => void,
    onError?: (error: Error) => void,
    onComplete?: () => void
  ): Promise<string> {
    const url = `${this.baseUrl}${servicePath}`;
    const streamId = `${servicePath}-${Date.now()}`;
    const controller = new AbortController();

    this.activeStreams.set(streamId, controller);

    try {
      const response = await fetch(url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Accept': 'text/plain', // For streaming responses
        },
        body: JSON.stringify(request),
        signal: controller.signal,
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const reader = response.body?.getReader();
      if (!reader) {
        throw new Error('No response body reader available');
      }

      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();

        if (done) {
          this.activeStreams.delete(streamId);
          if (onComplete) onComplete();
          break;
        }

        buffer += decoder.decode(value, { stream: true });

        // Process complete JSON lines
        const lines = buffer.split('\n');
        buffer = lines.pop() || ''; // Keep the incomplete line in buffer

        for (const line of lines) {
          if (line.trim()) {
            try {
              const message = JSON.parse(line);
              onMessage(message as T);
            } catch (parseError) {
              if (onError) onError(new Error(`Failed to parse streaming message: ${parseError}`));
            }
          }
        }
      }
    } catch (error) {
      this.activeStreams.delete(streamId);
      if (error instanceof Error && error.name === 'AbortError') {
        // Stream was cancelled, this is expected
        return streamId;
      }
      if (onError) onError(new Error(`Streaming request failed: ${error}`));
    }

    return streamId;
  }

  // Cancel a streaming request
  cancelStream(streamId: string): void {
    const controller = this.activeStreams.get(streamId);
    if (controller) {
      controller.abort();
      this.activeStreams.delete(streamId);
    }
  }

  // Database Service Methods
  async createConnection(request: CreateConnectionRequest): Promise<CreateConnectionResponse> {
    return this.makeRequest<CreateConnectionResponse>('/sqlstudio.database.DatabaseService/CreateConnection', request);
  }

  async getConnection(request: GetConnectionRequest): Promise<GetConnectionResponse> {
    return this.makeRequest<GetConnectionResponse>('/sqlstudio.database.DatabaseService/GetConnection', request);
  }

  async listConnections(request: ListConnectionsRequest): Promise<ListConnectionsResponse> {
    return this.makeRequest<ListConnectionsResponse>('/sqlstudio.database.DatabaseService/ListConnections', request);
  }

  async updateConnection(request: UpdateConnectionRequest): Promise<UpdateConnectionResponse> {
    return this.makeRequest<UpdateConnectionResponse>('/sqlstudio.database.DatabaseService/UpdateConnection', request);
  }

  async deleteConnection(request: DeleteConnectionRequest): Promise<DeleteConnectionResponse> {
    return this.makeRequest<DeleteConnectionResponse>('/sqlstudio.database.DatabaseService/DeleteConnection', request);
  }

  async testConnection(request: TestConnectionRequest): Promise<TestConnectionResponse> {
    return this.makeRequest<TestConnectionResponse>('/sqlstudio.database.DatabaseService/TestConnection', request);
  }

  async getSchemas(request: GetSchemasRequest): Promise<GetSchemasResponse> {
    return this.makeRequest<GetSchemasResponse>('/sqlstudio.database.DatabaseService/GetSchemas', request);
  }

  async getTables(request: GetTablesRequest): Promise<GetTablesResponse> {
    return this.makeRequest<GetTablesResponse>('/sqlstudio.database.DatabaseService/GetTables', request);
  }

  async getTableMetadata(request: GetTableMetadataRequest): Promise<GetTableMetadataResponse> {
    return this.makeRequest<GetTableMetadataResponse>('/sqlstudio.database.DatabaseService/GetTableMetadata', request);
  }

  // Streaming connection health
  async subscribeToConnectionHealth(
    request: GetConnectionHealthRequest,
    onMessage: (response: ConnectionHealthResponse) => void,
    onError?: (error: Error) => void,
    onComplete?: () => void
  ): Promise<string> {
    return this.makeStreamingRequest<ConnectionHealthResponse>(
      '/sqlstudio.database.DatabaseService/GetConnectionHealth',
      request,
      onMessage,
      onError,
      onComplete
    );
  }

  // Query Service Methods
  async executeQuery(request: ExecuteQueryRequest): Promise<ExecuteQueryResponse> {
    return this.makeRequest<ExecuteQueryResponse>('/sqlstudio.query.QueryService/ExecuteQuery', request);
  }

  // Streaming query execution
  async executeStreamingQuery(
    request: ExecuteStreamingQueryRequest,
    onMessage: (response: StreamingQueryResponse) => void,
    onError?: (error: Error) => void,
    onComplete?: () => void
  ): Promise<string> {
    return this.makeStreamingRequest<StreamingQueryResponse>(
      '/sqlstudio.query.QueryService/ExecuteStreamingQuery',
      request,
      onMessage,
      onError,
      onComplete
    );
  }

  async cancelQuery(request: CancelQueryRequest): Promise<CancelQueryResponse> {
    return this.makeRequest<CancelQueryResponse>('/sqlstudio.query.QueryService/CancelQuery', request);
  }

  async getQueryStatus(request: GetQueryStatusRequest): Promise<GetQueryStatusResponse> {
    return this.makeRequest<GetQueryStatusResponse>('/sqlstudio.query.QueryService/GetQueryStatus', request);
  }

  async listActiveQueries(request: ListActiveQueriesRequest): Promise<ListActiveQueriesResponse> {
    return this.makeRequest<ListActiveQueriesResponse>('/sqlstudio.query.QueryService/ListActiveQueries', request);
  }

  async executeBatchEdit(request: ExecuteBatchEditRequest): Promise<ExecuteBatchEditResponse> {
    return this.makeRequest<ExecuteBatchEditResponse>('/sqlstudio.query.QueryService/ExecuteBatchEdit', request);
  }

  async getQueryHistory(request: GetQueryHistoryRequest): Promise<GetQueryHistoryResponse> {
    return this.makeRequest<GetQueryHistoryResponse>('/sqlstudio.query.QueryService/GetQueryHistory', request);
  }

  async explainQuery(request: ExplainQueryRequest): Promise<ExplainQueryResponse> {
    return this.makeRequest<ExplainQueryResponse>('/sqlstudio.query.QueryService/ExplainQuery', request);
  }
}

// Create singleton instance
export const grpcWebClient = new GrpcWebClient();

// Export types for easy importing
export type {
  CancelQueryRequest,
  CancelQueryResponse,
  ConnectionHealthResponse,
  // Database types
  CreateConnectionRequest,
  CreateConnectionResponse,
  DeleteConnectionRequest,
  DeleteConnectionResponse,
  ExecuteBatchEditRequest,
  ExecuteBatchEditResponse,
  // Query types
  ExecuteQueryRequest,
  ExecuteQueryResponse,
  ExecuteStreamingQueryRequest,
  ExplainQueryRequest,
  ExplainQueryResponse,
  GetConnectionHealthRequest,
  GetConnectionRequest,
  GetConnectionResponse,
  GetQueryHistoryRequest,
  GetQueryHistoryResponse,
  GetQueryStatusRequest,
  GetQueryStatusResponse,
  GetSchemasRequest,
  GetSchemasResponse,
  GetTableMetadataRequest,
  GetTableMetadataResponse,
  GetTablesRequest,
  GetTablesResponse,
  ListActiveQueriesRequest,
  ListActiveQueriesResponse,
  ListConnectionsRequest,
  ListConnectionsResponse,
  StreamingQueryResponse,
  TestConnectionRequest,
  TestConnectionResponse,
  UpdateConnectionRequest,
  UpdateConnectionResponse,
};