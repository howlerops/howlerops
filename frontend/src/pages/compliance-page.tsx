/**
 * CompliancePage - Data Management & Compliance Dashboard
 *
 * Provides enterprise customers with tools to manage:
 * - Data retention policies
 * - GDPR compliance (data export/deletion)
 * - PII detection and masking
 * - Backup management
 * - Audit log viewing
 */

import {
  Alert,
  AlertDescription,
  AlertIcon,
  AlertTitle,
  Badge,
  Box,
  Button,
  Card,
  CardBody,
  CardHeader,
  Heading,
  HStack,
  Spinner,
  Tab,
  Table,
  TabList,
  TabPanel,
  TabPanels,
  Tabs,
  Tbody,
  Td,
  Text,
  Th,
  Thead,
  Tr,
  useDisclosure,
  useToast,
  VStack,
} from "@chakra-ui/react";
import React, { useCallback,useEffect, useState } from "react";

interface RetentionPolicy {
  id: string;
  resource_type: string;
  retention_days: number;
  auto_archive: boolean;
  archive_location?: string;
}

interface GDPRRequest {
  id: string;
  request_type: "export" | "delete";
  status: "pending" | "processing" | "completed" | "failed";
  requested_at: string;
  completed_at?: string;
  export_url?: string;
  error_message?: string;
}

interface PIIField {
  id: string;
  table_name: string;
  field_name: string;
  pii_type: string;
  confidence_score: number;
  verified: boolean;
}

// =============================================================================
// DEMO DATA - Replace with actual Wails API calls when backend is implemented
// These demonstrate the compliance features available in Howlerops Enterprise
// =============================================================================

const DEMO_RETENTION_POLICIES: RetentionPolicy[] = [
  {
    id: "rp-1",
    resource_type: "Query History",
    retention_days: 90,
    auto_archive: true,
    archive_location: "local_archive",
  },
  {
    id: "rp-2",
    resource_type: "Audit Logs",
    retention_days: 365,
    auto_archive: true,
    archive_location: "local_archive",
  },
  {
    id: "rp-3",
    resource_type: "Connection Logs",
    retention_days: 30,
    auto_archive: false,
  },
  {
    id: "rp-4",
    resource_type: "Export Files",
    retention_days: 7,
    auto_archive: false,
  },
];

const DEMO_GDPR_REQUESTS: GDPRRequest[] = [
  {
    id: "gdpr-1",
    request_type: "export",
    status: "completed",
    requested_at: new Date(Date.now() - 7 * 24 * 60 * 60 * 1000).toISOString(),
    completed_at: new Date(Date.now() - 6 * 24 * 60 * 60 * 1000).toISOString(),
    export_url: "#demo-export",
  },
  {
    id: "gdpr-2",
    request_type: "export",
    status: "processing",
    requested_at: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
  },
];

const DEMO_PII_FIELDS: PIIField[] = [
  {
    id: "pii-1",
    table_name: "users",
    field_name: "email",
    pii_type: "EMAIL",
    confidence_score: 0.98,
    verified: true,
  },
  {
    id: "pii-2",
    table_name: "users",
    field_name: "phone_number",
    pii_type: "PHONE",
    confidence_score: 0.95,
    verified: true,
  },
  {
    id: "pii-3",
    table_name: "customers",
    field_name: "ssn",
    pii_type: "SSN",
    confidence_score: 0.99,
    verified: true,
  },
  {
    id: "pii-4",
    table_name: "orders",
    field_name: "billing_address",
    pii_type: "ADDRESS",
    confidence_score: 0.87,
    verified: false,
  },
  {
    id: "pii-5",
    table_name: "employees",
    field_name: "date_of_birth",
    pii_type: "DOB",
    confidence_score: 0.92,
    verified: false,
  },
];

// =============================================================================
// END DEMO DATA
// =============================================================================

export const CompliancePage: React.FC = () => {
  const [retentionPolicies, setRetentionPolicies] = useState<RetentionPolicy[]>(
    []
  );
  const [gdprRequests, setGDPRRequests] = useState<GDPRRequest[]>([]);
  const [piiFields, setPIIFields] = useState<PIIField[]>([]);
  const [loading, setLoading] = useState(true);
  const [isDemo, setIsDemo] = useState(false);
  const toast = useToast();
  const { _isOpen, onOpen, _onClose } = useDisclosure();

  const loadRetentionPolicies = useCallback(async () => {
    try {
      // When backend compliance API is implemented, replace with:
      // const result = await wailsEndpoints.compliance.getRetentionPolicies()
      // For now, use demo data to showcase the feature
      setRetentionPolicies(DEMO_RETENTION_POLICIES);
      setIsDemo(true);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "Unknown error";
      toast({
        title: "Error loading retention policies",
        description: errorMessage,
        status: "error",
        duration: 5000,
      });
    } finally {
      setLoading(false);
    }
  }, [toast]);

  // Load compliance data
  useEffect(() => {
    loadRetentionPolicies();
    loadGDPRRequests();
    loadPIIFields();
  }, [loadRetentionPolicies]);

  const loadGDPRRequests = async () => {
    try {
      // When backend compliance API is implemented, replace with:
      // const result = await wailsEndpoints.compliance.getGDPRRequests()
      // For now, use demo data to showcase the feature
      setGDPRRequests(DEMO_GDPR_REQUESTS);
    } catch (error) {
      console.error("Error loading GDPR requests:", error);
    }
  };

  const loadPIIFields = async () => {
    try {
      // When backend compliance API is implemented, replace with:
      // const result = await wailsEndpoints.compliance.getPIIFields()
      // For now, use demo data to showcase the feature
      setPIIFields(DEMO_PII_FIELDS);
    } catch (error) {
      console.error("Error loading PII fields:", error);
    }
  };

  const _createRetentionPolicy = async (policy: Partial<RetentionPolicy>) => {
    try {
      const orgId = "current-org-id";
      const response = await fetch(
        `/api/organizations/${orgId}/retention-policy`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(policy),
        }
      );
      const data = await response.json();
      setRetentionPolicies([...retentionPolicies, data]);
      toast({
        title: "Retention policy created",
        status: "success",
        duration: 3000,
      });
    } catch (error) {
      toast({
        title: "Error creating policy",
        description: error.message,
        status: "error",
        duration: 5000,
      });
    }
  };

  const requestDataExport = async () => {
    try {
      // When backend compliance API is implemented, replace with:
      // const result = await wailsEndpoints.compliance.requestDataExport()
      // For now, simulate the request with demo data
      const newRequest: GDPRRequest = {
        id: `gdpr-${Date.now()}`,
        request_type: "export",
        status: "pending",
        requested_at: new Date().toISOString(),
      };
      setGDPRRequests([newRequest, ...gdprRequests]);
      toast({
        title: isDemo ? "[Demo] Data export requested" : "Data export requested",
        description: isDemo
          ? "This is a demo - no actual export will be created"
          : "You will receive an email when the export is ready",
        status: "info",
        duration: 5000,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "Unknown error";
      toast({
        title: "Error requesting export",
        description: errorMessage,
        status: "error",
        duration: 5000,
      });
    }
  };

  const requestDataDeletion = async () => {
    if (
      !confirm(
        isDemo
          ? "This is a demo - no data will actually be deleted. Continue?"
          : "Are you sure you want to delete all your data? This action cannot be undone."
      )
    ) {
      return;
    }

    try {
      // When backend compliance API is implemented, replace with:
      // const result = await wailsEndpoints.compliance.requestDataDeletion()
      // For now, simulate the request with demo data
      const newRequest: GDPRRequest = {
        id: `gdpr-${Date.now()}`,
        request_type: "delete",
        status: "pending",
        requested_at: new Date().toISOString(),
      };
      setGDPRRequests([newRequest, ...gdprRequests]);
      toast({
        title: isDemo ? "[Demo] Data deletion requested" : "Data deletion requested",
        description: isDemo
          ? "This is a demo - no data will be deleted"
          : "Your account and all data will be deleted within 30 days",
        status: "warning",
        duration: 5000,
      });
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : "Unknown error";
      toast({
        title: "Error requesting deletion",
        description: errorMessage,
        status: "error",
        duration: 5000,
      });
    }
  };

  if (loading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        height="400px"
      >
        <Spinner size="xl" />
      </Box>
    );
  }

  return (
    <Box p={6}>
      <VStack spacing={6} align="stretch">
        {isDemo && (
          <Alert status="warning" variant="left-accent">
            <AlertIcon />
            <Box>
              <AlertTitle>Demo Mode</AlertTitle>
              <AlertDescription>
                Compliance features are shown with sample data. Connect this page
                to the backend compliance API to manage real retention policies,
                GDPR requests, and PII detection.
              </AlertDescription>
            </Box>
          </Alert>
        )}

        <Box>
          <Heading size="lg" mb={2}>
            Data Compliance & Management
          </Heading>
          <Text color="gray.600">
            Manage data retention, GDPR requests, and PII detection
          </Text>
        </Box>

        <Tabs>
          <TabList>
            <Tab>Retention Policies</Tab>
            <Tab>GDPR Requests</Tab>
            <Tab>PII Detection</Tab>
            <Tab>Audit Logs</Tab>
          </TabList>

          <TabPanels>
            {/* Retention Policies Tab */}
            <TabPanel>
              <VStack spacing={4} align="stretch">
                <HStack justify="space-between">
                  <Heading size="md">Data Retention Policies</Heading>
                  <Button colorScheme="blue" onClick={onOpen}>
                    Create Policy
                  </Button>
                </HStack>

                <Alert status="info">
                  <AlertIcon />
                  <Box>
                    <AlertTitle>Automatic Enforcement</AlertTitle>
                    <AlertDescription>
                      Retention policies are automatically enforced daily at 2
                      AM. Old data is archived before deletion.
                    </AlertDescription>
                  </Box>
                </Alert>

                <Table variant="simple">
                  <Thead>
                    <Tr>
                      <Th>Resource Type</Th>
                      <Th>Retention Period</Th>
                      <Th>Auto Archive</Th>
                      <Th>Actions</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {retentionPolicies.map((policy) => (
                      <Tr key={policy.id}>
                        <Td>
                          <Badge colorScheme="purple">
                            {policy.resource_type}
                          </Badge>
                        </Td>
                        <Td>{policy.retention_days} days</Td>
                        <Td>
                          <Badge
                            colorScheme={policy.auto_archive ? "green" : "gray"}
                          >
                            {policy.auto_archive ? "Enabled" : "Disabled"}
                          </Badge>
                        </Td>
                        <Td>
                          <HStack>
                            <Button size="sm">Edit</Button>
                            <Button size="sm" colorScheme="red" variant="ghost">
                              Delete
                            </Button>
                          </HStack>
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </VStack>
            </TabPanel>

            {/* GDPR Requests Tab */}
            <TabPanel>
              <VStack spacing={4} align="stretch">
                <Heading size="md">GDPR Data Requests</Heading>

                <Card>
                  <CardHeader>
                    <Heading size="sm">Request Your Data</Heading>
                  </CardHeader>
                  <CardBody>
                    <VStack spacing={3} align="stretch">
                      <Text>
                        You have the right to access and download all your
                        personal data, or request complete deletion of your
                        account.
                      </Text>
                      <HStack>
                        <Button colorScheme="blue" onClick={requestDataExport}>
                          Export My Data
                        </Button>
                        <Button
                          colorScheme="red"
                          variant="outline"
                          onClick={requestDataDeletion}
                        >
                          Delete My Account
                        </Button>
                      </HStack>
                    </VStack>
                  </CardBody>
                </Card>

                <Heading size="sm" mt={4}>
                  Request History
                </Heading>
                <Table variant="simple">
                  <Thead>
                    <Tr>
                      <Th>Type</Th>
                      <Th>Status</Th>
                      <Th>Requested</Th>
                      <Th>Actions</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {gdprRequests.map((request) => (
                      <Tr key={request.id}>
                        <Td>
                          <Badge
                            colorScheme={
                              request.request_type === "export" ? "blue" : "red"
                            }
                          >
                            {request.request_type}
                          </Badge>
                        </Td>
                        <Td>
                          <Badge
                            colorScheme={
                              request.status === "completed"
                                ? "green"
                                : request.status === "failed"
                                ? "red"
                                : request.status === "processing"
                                ? "yellow"
                                : "gray"
                            }
                          >
                            {request.status}
                          </Badge>
                        </Td>
                        <Td>
                          {new Date(request.requested_at).toLocaleDateString()}
                        </Td>
                        <Td>
                          {request.export_url && (
                            <Button
                              size="sm"
                              as="a"
                              href={request.export_url}
                              download
                            >
                              Download
                            </Button>
                          )}
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </VStack>
            </TabPanel>

            {/* PII Detection Tab */}
            <TabPanel>
              <VStack spacing={4} align="stretch">
                <Heading size="md">PII Detection & Masking</Heading>

                <Alert status="warning">
                  <AlertIcon />
                  <Box>
                    <AlertTitle>Automatic Detection</AlertTitle>
                    <AlertDescription>
                      Howlerops automatically detects and masks PII in query
                      results. You can review and verify detected fields below.
                    </AlertDescription>
                  </Box>
                </Alert>

                <Table variant="simple">
                  <Thead>
                    <Tr>
                      <Th>Table</Th>
                      <Th>Field</Th>
                      <Th>PII Type</Th>
                      <Th>Confidence</Th>
                      <Th>Verified</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {piiFields.map((field) => (
                      <Tr key={field.id}>
                        <Td>{field.table_name}</Td>
                        <Td>{field.field_name}</Td>
                        <Td>
                          <Badge colorScheme="orange">{field.pii_type}</Badge>
                        </Td>
                        <Td>{(field.confidence_score * 100).toFixed(0)}%</Td>
                        <Td>
                          {field.verified ? (
                            <Badge colorScheme="green">Verified</Badge>
                          ) : (
                            <Button size="sm">Verify</Button>
                          )}
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </VStack>
            </TabPanel>

            {/* Audit Logs Tab */}
            <TabPanel>
              <VStack spacing={4} align="stretch">
                <Heading size="md">Audit Logs</Heading>
                <Text color="gray.600">
                  View detailed audit trails of all data access and
                  modifications.
                </Text>
                <Alert status="info">
                  <AlertIcon />
                  Coming soon: Field-level change tracking and PII access logs
                </Alert>
              </VStack>
            </TabPanel>
          </TabPanels>
        </Tabs>
      </VStack>
    </Box>
  );
};

export default CompliancePage;
