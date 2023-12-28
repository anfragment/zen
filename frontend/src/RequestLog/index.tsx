import { CardList, Card, Tag, Collapse, HTMLTable } from '@blueprintjs/core';
import { useEffect, useState } from 'react';

// eslint-disable-next-line import/no-relative-packages
import { EventsOn } from '../../wailsjs/runtime';

import './index.css';

type Rule = {
  FilterName: string;
  RawRule: string;
};

type FilterActionKind = 'blocked' | 'redirected' | 'modified';

type FilterAction = {
  id: string;
  kind: FilterActionKind;
  method: string;
  url: string;
  to: string;
  referer: string;
  rules: Rule[];
  createdAt: Date;
};

export function RequestLog() {
  const [logs, setLogs] = useState<FilterAction[]>([]);

  useEffect(() => {
    const cancel = EventsOn('filter:action', (action: Omit<FilterAction, 'id' | 'createdAt'>) => {
      setLogs((logs) =>
        [
          {
            ...action,
            id: id(),
            createdAt: new Date(),
          },
          ...logs,
        ].slice(0, 200),
      );
    });

    return () => {
      cancel();
    };
  }, []);

  return (
    <div className="request-log">
      {logs.length === 0 ? (
        <p className="request-log__empty">Start browsing to see blocked requests.</p>
      ) : (
        <CardList compact>
          {logs.map((log) => (
            <RequestLogCard log={log} key={log.id} />
          ))}
        </CardList>
      )}
    </div>
  );
}

function RequestLogCard({ log }: { log: FilterAction }) {
  const [isOpen, setIsOpen] = useState(false);

  const { hostname } = new URL(log.url, 'http://foo'); // Setting the base url somehow helps with parsing //hostname:port URLs

  return (
    <>
      <Card key={log.id} className="request-log__card" interactive onClick={() => setIsOpen(!isOpen)}>
        <Tag minimal intent={log.kind === 'blocked' ? 'danger' : 'warning'}>
          {hostname}
        </Tag>
        <div className="bp5-text-muted">{log.createdAt.toLocaleTimeString([], { timeStyle: 'short' })}</div>
      </Card>

      <Collapse isOpen={isOpen}>
        <Card className="request-log__card__details">
          <p className="request-log__card__details__value">
            <strong>Method: </strong>
            <Tag minimal intent="primary">
              {log.method}
            </Tag>
          </p>
          <p className="request-log__card__details__value">
            <strong>URL: </strong>
            {log.url}
          </p>
          {log.kind === 'redirected' && (
            <p className="request-log__card__details__value">
              <strong>Redirected to: </strong>
              {log.to}
            </p>
          )}
          {log.referer && (
            <p className="request-log__card__details__value">
              <strong>Referer: </strong>
              {log.referer}
            </p>
          )}
          <HTMLTable bordered compact striped className="request-log__card__details__rules">
            <thead>
              <tr>
                <th>Filter name</th>
                <th>Rule</th>
              </tr>
            </thead>
            <tbody>
              {log.rules.map((rule) => (
                <tr key={rule.RawRule}>
                  <td>{rule.FilterName}</td>
                  <td>{rule.RawRule}</td>
                </tr>
              ))}
            </tbody>
          </HTMLTable>
        </Card>
      </Collapse>
    </>
  );
}

function id(): string {
  return Math.random().toString(36).slice(2, 9);
}
