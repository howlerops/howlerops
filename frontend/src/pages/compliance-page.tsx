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

import React, { useState, useEffect, useCallback } from "react";
import {
  Box,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Heading,
  Text,
  VStack,
  HStack,
  Button,
  Card,
  CardHeader,
  CardBody,
  Badge,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  useToast,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Spinner,
  useDisclosure,
} from "@chakra-ui/react";

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

export const CompliancePage: React.FC = () => {
  const [retentionPolicies, setRetentionPolicies] = useState<RetentionPolicy[]>(
    []
  );
  const [gdprRequests, setGDPRRequests] = useState<GDPRRequest[]>([]);
  const [piiFields, setPIIFields] = useState<PIIField[]>([]);
  const [loading, setLoading] = useState(true);
  const toast = useToast();
  const { _isOpen, onOpen, _onClose } = useDisclosure();

  const loadRetentionPolicies = useCallback(async () => {
    try {
      // TODO: Replace with actual API call
      const orgId = "current-org-id";
      const response = await fetch(
        `/api/organizations/${orgId}/retention-policy`
      );
      const data = await response.json();
      setRetentionPolicies(data);
    } catch (error) {
      toast({
        title: "Error loading retention policies",
        description: error.message,
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
      // TODO: Replace with actual API call
      const response = await fetch("/api/gdpr/requests");
      const data = await response.json();
      setGDPRRequests(data);
    } catch (error) {
      console.error("Error loading GDPR requests:", error);
    }
  };

  const loadPIIFields = async () => {
    try {
      const response = await fetch("/api/pii/fields");
      const data = await response.json();
      setPIIFields(data);
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
      const response = await fetch("/api/gdpr/export", { method: "POST" });
      const data = await response.json();
      setGDPRRequests([data, ...gdprRequests]);
      toast({
        title: "Data export requested",
        description: "You will receive an email when the export is ready",
        status: "info",
        duration: 5000,
      });
    } catch (error) {
      toast({
        title: "Error requesting export",
        description: error.message,
        status: "error",
        duration: 5000,
      });
    }
  };

  const requestDataDeletion = async () => {
    if (
      !confirm(
        "Are you sure you want to delete all your data? This action cannot be undone."
      )
    ) {
      return;
    }

    try {
      const response = await fetch("/api/gdpr/delete", { method: "POST" });
      const data = await response.json();
      setGDPRRequests([data, ...gdprRequests]);
      toast({
        title: "Data deletion requested",
        description: "Your account and all data will be deleted within 30 days",
        status: "warning",
        duration: 5000,
      });
    } catch (error) {
      toast({
        title: "Error requesting deletion",
        description: error.message,
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
